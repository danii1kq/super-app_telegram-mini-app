package chat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/danii1kq/srunal/internal/core/auth"
	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

func attachmentTestService(t *testing.T) *Service {
	t.Helper()
	s := testChatService(t)
	migration, err := os.ReadFile("../../../../db/migrations/000011_chat_attachments.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	return s
}

func attachmentPhoto(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestPrepareAttachment(t *testing.T) {
	raw := []byte("<html>private file contents</html>")
	info, data, err := prepareAttachment("file", "C:\\private\\folder\\report\r\n\u202e:2026.html", raw)
	sum := sha256.Sum256(raw)
	if err != nil || !bytes.Equal(data, raw) || info.Name != "report2026.html" || info.Mime != "application/octet-stream" || info.Size != len(raw) || info.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("file preparation: %+v, %q, %v", info, data, err)
	}
	for _, test := range []struct {
		kind string
		data []byte
	}{
		{"file", nil}, {"file", make([]byte, MaxAttachmentBytes+1)},
		{"video", raw}, {"photo", raw},
	} {
		if _, _, err := prepareAttachment(test.kind, "file", test.data); err == nil {
			t.Errorf("accepted invalid %q of %d bytes", test.kind, len(test.data))
		}
	}
	photo := attachmentPhoto(t)
	metadata := []byte("Exif\x00\x00PRIVATE GPS LOCATION")
	photo = append(append(append(append([]byte{}, photo[:2]...), 0xff, 0xe1, 0, byte(len(metadata)+2)), metadata...), photo[2:]...)
	info, data, err = prepareAttachment("photo", "location.jpg", photo)
	if err != nil || info.Mime != "image/jpeg" || info.Name != "photo.jpg" || info.Width != 32 || info.Height != 24 || bytes.Contains(data, metadata) {
		t.Fatalf("normalized photo retained metadata or changed dimensions: %+v, %v", info, err)
	}
	var pngImage bytes.Buffer
	if err := png.Encode(&pngImage, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	oversize := pngImage.Bytes()
	binary.BigEndian.PutUint32(oversize[16:20], 5000)
	binary.BigEndian.PutUint32(oversize[20:24], 4000)
	binary.BigEndian.PutUint32(oversize[29:33], crc32.ChecksumIEEE(oversize[12:29]))
	if _, _, err := prepareAttachment("photo", "bomb.png", oversize); err == nil {
		t.Fatal("accepted photo above pixel budget")
	}
}

func TestAttachmentStorageAccessAndReplay(t *testing.T) {
	s := attachmentTestService(t)
	ctx := context.Background()
	conversation, err := s.OpenDirect(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("PRIVATE_FILE_CONTENTS_873f0748")
	message, err := s.SendAttachment(ctx, 1, conversation.ID, "attachment-first", " Caption ", "file", "private-report.txt", raw)
	if err != nil || message.Attachment == nil || message.Text != "Caption" {
		t.Fatalf("send file: %+v, %v", message, err)
	}
	var sealed, payload []byte
	var keyID string
	if err := s.pool.QueryRow(ctx, `SELECT a.body_sealed, a.key_id, m.payload_sealed FROM chat_attachments a JOIN chat_messages m ON m.id = a.message_id WHERE m.id = $1`, message.ID).Scan(&sealed, &keyID, &payload); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, raw) || bytes.Contains(payload, []byte("private-report.txt")) || bytes.Contains(payload, []byte("Caption")) {
		t.Fatal("file or message metadata stored in plaintext")
	}
	if _, err := s.content.Open(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, attachmentAAD(conversation.ID, message.ID+1)); !errors.Is(err, secure.ErrDecrypt) {
		t.Fatalf("file was not bound to its message: %v", err)
	}
	data, info, err := s.DownloadAttachment(ctx, 2, conversation.ID, message.ID)
	if err != nil || !bytes.Equal(data, raw) || !reflect.DeepEqual(info, message.Attachment) {
		t.Fatalf("download changed bytes or metadata: %+v, %v", info, err)
	}
	history, _, err := s.History(ctx, 2, conversation.ID, HistoryQuery{})
	if err != nil || len(history) != 1 || !reflect.DeepEqual(history[0].Attachment, info) {
		t.Fatalf("history lost attachment: %+v, %v", history, err)
	}
	list, err := s.List(ctx, 2)
	if err != nil || len(list) != 1 || list[0].LastMessage == nil || !reflect.DeepEqual(list[0].LastMessage.Attachment, info) {
		t.Fatalf("preview lost attachment: %+v, %v", list, err)
	}
	replay, err := s.SendAttachment(ctx, 1, conversation.ID, "attachment-first", "Caption", "file", "private-report.txt", raw)
	if err != nil || replay.ID != message.ID || !replay.CreatedAt.Equal(message.CreatedAt) {
		t.Fatalf("attachment replay created another message: %+v, %v", replay, err)
	}
	for _, test := range []struct {
		caption, name string
		data          []byte
	}{
		{"Changed", "private-report.txt", raw}, {"Caption", "renamed.txt", raw}, {"Caption", "private-report.txt", []byte("changed contents")},
	} {
		_, err := s.SendAttachment(ctx, 1, conversation.ID, "attachment-first", test.caption, "file", test.name, test.data)
		assertChatStatus(t, err, 409)
	}
	_, err = s.Send(ctx, 1, conversation.ID, "attachment-first", "Caption")
	assertChatStatus(t, err, 409)
	_, _, err = s.DownloadAttachment(ctx, 3, conversation.ID, message.ID)
	assertChatStatus(t, err, 404)
	_, err = s.SendAttachment(ctx, 3, conversation.ID, "outsider-upload", "", "file", "private.txt", raw)
	assertChatStatus(t, err, 404)
	other, err := s.OpenDirect(ctx, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.DownloadAttachment(ctx, 1, other.ID, message.ID)
	assertChatStatus(t, err, 404)
	tampered := append([]byte{}, sealed...)
	tampered[len(tampered)-1] ^= 1
	if _, err := s.pool.Exec(ctx, `UPDATE chat_attachments SET body_sealed = $2 WHERE message_id = $1`, message.ID, tampered); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.DownloadAttachment(ctx, 2, conversation.ID, message.ID); !errors.Is(err, secure.ErrDecrypt) {
		t.Fatalf("tampered ciphertext was accepted: %v", err)
	}
	privateFile, err := s.SendAttachment(ctx, 1, other.ID, "revoked-file", "", "file", "private.txt", raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM chat_members WHERE conversation_id = $1 AND user_id = 4`, other.ID); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.DownloadAttachment(ctx, 4, other.ID, privateFile.ID)
	assertChatStatus(t, err, 404)
}

func TestAttachmentQuota(t *testing.T) {
	for _, limit := range []struct {
		name        string
		count, size int
	}{{"bytes", 10, 20 << 20}, {"count", 1000, 1}} {
		t.Run(limit.name, func(t *testing.T) {
			s := attachmentTestService(t)
			ctx := context.Background()
			conversation, err := s.OpenDirect(ctx, 1, 2)
			if err != nil {
				t.Fatal(err)
			}
			// Only accounting rows are needed; no large file allocation belongs in a quota test.
			if _, err := s.pool.Exec(ctx, `WITH messages AS (
			 INSERT INTO chat_messages (conversation_id,sender_id,seq,client_message_id,payload_sealed,key_id)
			 SELECT $1,1,100+n,'quota-' || n,'x'::bytea,'test' FROM generate_series(1,$2::int) n RETURNING id)
			 INSERT INTO chat_attachments (message_id,sender_id,byte_size,body_sealed,key_id) SELECT id,1,$3,'x'::bytea,'test' FROM messages`, conversation.ID, limit.count, limit.size); err != nil {
				t.Fatal(err)
			}
			_, err = s.SendAttachment(ctx, 1, conversation.ID, "over-quota", "", "file", "more.txt", []byte("more"))
			assertChatStatus(t, err, 400)
			var quotaErr *httpapi.Error
			if !errors.As(err, &quotaErr) || quotaErr.Code != "storage_limit" {
				t.Fatalf("expected storage quota rejection, got %v", err)
			}
			var seq int64
			if err := s.pool.QueryRow(ctx, `SELECT last_message_seq FROM chat_conversations WHERE id = $1`, conversation.ID).Scan(&seq); err != nil || seq != 0 {
				t.Fatalf("rejected file advanced message sequence: %d, %v", seq, err)
			}
		})
	}
}

func TestAttachmentTransferSlots(t *testing.T) {
	s := attachmentTestService(t)
	ctx := context.Background()
	conversation, err := s.OpenDirect(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,1,now()+interval '1 hour')`, secure.TokenHash("file-owner")); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandler(s, auth.NewService(s.pool, nil, nil, time.Hour), NewHub()).registerAttachments(mux)
	for len(s.uploadSlots) < cap(s.uploadSlots) {
		s.uploadSlots <- struct{}{}
	}
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/conversations/%d/attachments", conversation.ID), nil)
	request.Header.Set("Authorization", "Bearer file-owner")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("upload concurrency was not bounded: %d", response.Code)
	}
	for len(s.uploadSlots) > 0 {
		<-s.uploadSlots
	}
	for len(s.downloadSlots) < cap(s.downloadSlots) {
		s.downloadSlots <- struct{}{}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(cancelled)
	response = httptest.NewRecorder()
	NewHandler(s, nil, nil).downloadAttachment(response, request)
	if response.Body.Len() != 0 || len(s.downloadSlots) != cap(s.downloadSlots) {
		t.Fatal("cancelled queued download allocated a slot or wrote bytes")
	}
	for len(s.downloadSlots) > 0 {
		<-s.downloadSlots
	}
}

func TestAttachmentHTTP(t *testing.T) {
	s := attachmentTestService(t)
	ctx := context.Background()
	conversation, err := s.OpenDirect(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	for id, token := range map[int64]string{1: "file-owner", 2: "file-member", 3: "file-outsider"} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_hash,user_id,expires_at) VALUES ($1,$2,now()+interval '1 hour')`, secure.TokenHash(token), id); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	NewHandler(s, auth.NewService(s.pool, nil, nil, time.Hour), NewHub()).registerAttachments(mux)
	file, err := s.SendAttachment(ctx, 1, conversation.ID, "http-file", "", "file", "report.html", []byte("<script>alert('unsafe')</script>"))
	if err != nil {
		t.Fatal(err)
	}
	photo, err := s.SendAttachment(ctx, 1, conversation.ID, "http-photo", "", "photo", "photo.jpg", attachmentPhoto(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		message           Message
		token             string
		status            int
		mime, disposition string
	}{
		{file, "file-member", 200, "application/octet-stream", "attachment"},
		{photo, "file-member", 200, "image/jpeg", "inline"},
		{file, "file-outsider", 404, "", ""}, {file, "", 401, "", ""},
	} {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/conversations/%d/messages/%d/attachment", conversation.ID, test.message.ID), nil)
		if test.token != "" {
			req.Header.Set("Authorization", "Bearer "+test.token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != test.status {
			t.Fatalf("download status %d, want %d: %s", response.Code, test.status, response.Body.String())
		}
		if test.status == 200 {
			if response.Header().Get("Content-Type") != test.mime || !strings.HasPrefix(response.Header().Get("Content-Disposition"), test.disposition) || response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" {
				t.Fatalf("unsafe attachment response headers: %v", response.Header())
			}
		} else if bytes.Contains(response.Body.Bytes(), []byte("<script>")) {
			t.Fatal("private bytes leaked in failed response")
		}
	}
	for _, test := range []struct {
		name   string
		fields [][2]string
		files  int
		data   []byte
	}{
		{"valid", [][2]string{{"kind", "file"}, {"client_message_id", "http-upload"}, {"text", "Caption"}}, 1, []byte("uploaded")},
		{"unknown field", [][2]string{{"kind", "file"}, {"unknown", "ignored"}, {"client_message_id", "unknown"}}, 1, []byte("file")},
		{"duplicate field", [][2]string{{"kind", "file"}, {"kind", "photo"}, {"client_message_id", "duplicate-field"}}, 1, []byte("file")},
		{"multiple files", [][2]string{{"kind", "file"}, {"client_message_id", "duplicate-file"}}, 2, []byte("file")},
		{"missing file", [][2]string{{"kind", "file"}, {"client_message_id", "missing-file"}}, 0, nil},
		{"corrupt photo", [][2]string{{"kind", "photo"}, {"client_message_id", "corrupt-photo"}}, 1, []byte("not a picture")},
		{"oversize", [][2]string{{"kind", "file"}, {"client_message_id", "oversize"}}, 1, make([]byte, MaxAttachmentBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for _, field := range test.fields {
				if err := writer.WriteField(field[0], field[1]); err != nil {
					t.Fatal(err)
				}
			}
			for n := 0; n < test.files; n++ {
				part, err := writer.CreateFormFile("file", "upload.txt")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := part.Write(test.data); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/conversations/%d/attachments", conversation.ID), &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", "Bearer file-owner")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			want := 400
			if test.name == "valid" {
				want = 201
			}
			if response.Code != want {
				t.Fatalf("multipart status %d, want %d: %s", response.Code, want, response.Body.String())
			}
		})
	}
}
