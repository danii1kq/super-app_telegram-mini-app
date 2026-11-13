package app

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danii1kq/srunal/internal/platform/config"
)

func TestRetiredChatRoutesAreUnavailable(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	handler, err := NewAPI(context.Background(), config.Config{ContentKey: key, IdentityKey: key, HMACIndexKey: key, DeliveryKey: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []struct{ method, path string }{
		{"GET", "/api/v1/conversations"}, {"POST", "/api/v1/conversations"},
		{"POST", "/api/v1/conversations/groups"}, {"GET", "/api/v1/conversations/1/messages"},
		{"POST", "/api/v1/conversations/1/attachments"}, {"GET", "/api/v1/events/stream?conversation=1"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(request.method, request.path, nil))
		if response.Code != 404 || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("retired %s %s returned %d", request.method, request.path, response.Code)
		}
	}
}

func TestImportedAppsAreRegisteredBehindAuthentication(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	handler, err := NewAPI(context.Background(), config.Config{ContentKey: key, IdentityKey: key, HMACIndexKey: key, DeliveryKey: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/tinder/profile", "/api/v1/farm", "/api/v1/aura"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 401 {
			t.Fatalf("%s returned %d", path, w.Code)
		}
	}
}
