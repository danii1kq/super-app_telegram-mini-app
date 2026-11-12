// Package notes owns SruNotes: private text notes and read grants.
//
// A note is visible to its owner and to recipients holding an active grant.
// Every read re-checks the grant, so revoking or deleting closes access at once.
// The card in a chat stores the note id, never a copy of the text.
package notes

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
	"github.com/danii1kq/srunal/internal/platform/postgres"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

type Service struct {
	pool    *postgres.Pool
	content *secure.Cipher
}

func NewService(pool *postgres.Pool, content *secure.Cipher) *Service {
	return &Service{pool: pool, content: content}
}

type Note struct {
	ID        int64
	OwnerID   int64
	Title     string
	Body      string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	// Mine is true when the viewer is the author.
	Mine bool
}

// List returns the viewer's own notes and the notes shared with them.
func (s *Service) List(ctx context.Context, viewerID int64) ([]Note, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT n.id, n.owner_id, n.title_sealed, n.body_sealed, n.key_id,
		        n.version, n.created_at, n.updated_at, n.owner_id = $1 AS mine
		   FROM notes n
		  WHERE n.deleted_at IS NULL
		    AND (
		         n.owner_id = $1
		         OR EXISTS (
		              SELECT 1 FROM note_grants g
		               WHERE g.note_id = n.id AND g.recipient_id = $1 AND g.state = 'active'
		                 AND (
		                      g.context_kind <> 'community'
		                      OR EXISTS (SELECT 1 FROM social_members m
		                                  WHERE m.community_id = g.context_id
		                                    AND m.user_id = $1 AND m.state = 'active')
		                 )
		         )
		    )
		  ORDER BY n.updated_at DESC, n.id DESC
		  LIMIT 200`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Note, 0, 16)
	for index := 0; rows.Next(); index++ {
		var note Note
		var titleSealed, bodySealed []byte
		var keyID string
		if err := rows.Scan(&note.ID, &note.OwnerID, &titleSealed, &bodySealed, &keyID,
			&note.Version, &note.CreatedAt, &note.UpdatedAt, &note.Mine,
		); err != nil {
			return nil, err
		}
		title, err := s.openTitle(titleSealed, keyID, note.ID, note.OwnerID)
		if err != nil {
			return nil, err
		}
		note.Title = title
		// The list shows the title only: the body is opened on the note screen.
		out = append(out, note)
	}
	return out, rows.Err()
}

// Get returns one note with its body, re-checking access.
func (s *Service) Get(ctx context.Context, viewerID, noteID int64) (Note, error) {
	var note Note
	var titleSealed, bodySealed []byte
	var keyID string
	err := s.pool.QueryRow(ctx,
		`SELECT n.id, n.owner_id, n.title_sealed, n.body_sealed, n.key_id,
		        n.version, n.created_at, n.updated_at, n.owner_id = $1 AS mine
		   FROM notes n
		  WHERE n.id = $2 AND n.deleted_at IS NULL
		    AND (
		         n.owner_id = $1
		         OR EXISTS (
		              SELECT 1 FROM note_grants g
		               WHERE g.note_id = n.id AND g.recipient_id = $1 AND g.state = 'active'
		                 AND (
		                      g.context_kind <> 'community'
		                      OR EXISTS (SELECT 1 FROM social_members m
		                                  WHERE m.community_id = g.context_id
		                                    AND m.user_id = $1 AND m.state = 'active')
		                 )
		         )
		    )`, viewerID, noteID,
	).Scan(&note.ID, &note.OwnerID, &titleSealed, &bodySealed, &keyID,
		&note.Version, &note.CreatedAt, &note.UpdatedAt, &note.Mine)
	if errors.Is(err, pgx.ErrNoRows) {
		// Neutral answer: a stranger learns nothing about the note.
		return Note{}, httpapi.NotFound("Заметка не найдена")
	}
	if err != nil {
		return Note{}, err
	}

	title, err := s.openTitle(titleSealed, keyID, note.ID, note.OwnerID)
	if err != nil {
		return Note{}, err
	}
	body, err := s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: bodySealed}, bodyAAD(note.ID, note.OwnerID))
	if err != nil {
		return Note{}, err
	}
	note.Title = title
	note.Body = body
	return note, nil
}

type NoteInput struct {
	Title string
	Body  string
}

// Create stores a new note owned by the author.
func (s *Service) Create(ctx context.Context, ownerID int64, input NoteInput) (Note, error) {
	if input.Body == "" {
		return Note{}, httpapi.Validation(map[string]string{"body": "Текст заметки не может быть пустым"}, "Проверьте поля")
	}
	if len([]rune(input.Body)) > 20000 {
		return Note{}, httpapi.Validation(map[string]string{"body": "Слишком длинный текст"}, "Проверьте поля")
	}

	var note Note
	err := s.pool.InTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO notes (owner_id, body_sealed, key_id) VALUES ($1, ''::bytea, 'pending')
			 RETURNING id, version, created_at, updated_at`, ownerID,
		).Scan(&note.ID, &note.Version, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return err
		}

		bodySealed, err := s.content.SealString(input.Body, bodyAAD(note.ID, ownerID))
		if err != nil {
			return err
		}
		var titleSealed *secure.Sealed
		if input.Title != "" {
			sealed, err := s.content.SealString(input.Title, titleAAD(note.ID, ownerID))
			if err != nil {
				return err
			}
			titleSealed = &sealed
		}

		if titleSealed != nil {
			_, err = tx.Exec(ctx,
				`UPDATE notes SET body_sealed = $2, key_id = $3, title_sealed = $4 WHERE id = $1`,
				note.ID, bodySealed.Ciphertext, bodySealed.KeyID, titleSealed.Ciphertext)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE notes SET body_sealed = $2, key_id = $3 WHERE id = $1`,
				note.ID, bodySealed.Ciphertext, bodySealed.KeyID)
		}
		return err
	})
	if err != nil {
		return Note{}, err
	}

	note.OwnerID = ownerID
	note.Title = input.Title
	note.Body = input.Body
	note.Mine = true
	return note, nil
}

// Update replaces the text, checking the version so a stale editor cannot
// silently overwrite a newer note.
func (s *Service) Update(ctx context.Context, ownerID, noteID int64, input NoteInput, version int) (Note, error) {
	if input.Body == "" {
		return Note{}, httpapi.Validation(map[string]string{"body": "Текст заметки не может быть пустым"}, "Проверьте поля")
	}

	bodySealed, err := s.content.SealString(input.Body, bodyAAD(noteID, ownerID))
	if err != nil {
		return Note{}, err
	}
	titleSealed, err := s.content.SealString(input.Title, titleAAD(noteID, ownerID))
	if err != nil {
		return Note{}, err
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE notes
		    SET body_sealed = $3, title_sealed = $4, key_id = $5,
		        version = version + 1, updated_at = now()
		  WHERE id = $1 AND owner_id = $2 AND version = $6 AND deleted_at IS NULL`,
		noteID, ownerID, bodySealed.Ciphertext, titleSealed.Ciphertext, bodySealed.KeyID, version)
	if err != nil {
		return Note{}, err
	}
	if tag.RowsAffected() == 0 {
		return Note{}, httpapi.Conflict("version_conflict", "Заметка изменена в другом месте")
	}
	return s.Get(ctx, ownerID, noteID)
}

// Delete soft-deletes the owner's note and revokes its grants.
func (s *Service) Delete(ctx context.Context, ownerID, noteID int64) error {
	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE notes SET deleted_at = now()
			  WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`, noteID, ownerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpapi.NotFound("Заметка не найдена")
		}
		_, err = tx.Exec(ctx,
			`UPDATE note_grants SET state = 'revoked' WHERE note_id = $1 AND state = 'active'`, noteID)
		return err
	})
}

// Grant shares the note with the listed recipients. Only the owner may grant.
func (s *Service) Grant(ctx context.Context, ownerID, noteID int64, recipients []int64, contextKind string, contextID *int64) error {
	if len(recipients) == 0 {
		return httpapi.Validation(map[string]string{"recipients": "Выберите получателей"}, "Проверьте поля")
	}
	if contextKind != "direct" && contextKind != "chat" && contextKind != "community" {
		contextKind = "direct"
	}

	return s.pool.InTx(ctx, func(tx pgx.Tx) error {
		var owner int64
		if err := tx.QueryRow(ctx,
			`SELECT owner_id FROM notes WHERE id = $1 AND deleted_at IS NULL`, noteID,
		).Scan(&owner); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpapi.NotFound("Заметка не найдена")
			}
			return err
		}
		if owner != ownerID {
			return httpapi.Forbidden("Делиться может только автор")
		}

		for _, recipient := range recipients {
			if recipient == ownerID {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO note_grants (note_id, recipient_id, granted_by, context_kind, context_id)
				 VALUES ($1, $2, $3, $4, $5)
				 ON CONFLICT (note_id, recipient_id)
				 DO UPDATE SET state = 'active', context_kind = excluded.context_kind,
				               context_id = excluded.context_id`,
				noteID, recipient, ownerID, contextKind, contextID,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// Revoke closes a grant.
func (s *Service) Revoke(ctx context.Context, ownerID, noteID, recipientID int64) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE note_grants SET state = 'revoked'
		  WHERE note_id = $1 AND recipient_id = $2 AND granted_by = $3 AND state = 'active'`,
		noteID, recipientID, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpapi.NotFound("Разрешение не найдено")
	}
	return nil
}

// Recipients lists who currently can read the note.
func (s *Service) Recipients(ctx context.Context, ownerID, noteID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT g.recipient_id
		   FROM note_grants g
		   JOIN notes n ON n.id = g.note_id
		  WHERE g.note_id = $1 AND n.owner_id = $2 AND g.state = 'active'`, noteID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int64, 0, 4)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) openTitle(sealed []byte, keyID string, noteID, ownerID int64) (string, error) {
	if len(sealed) == 0 {
		return "", nil
	}
	return s.content.OpenString(secure.Sealed{KeyID: keyID, Ciphertext: sealed}, titleAAD(noteID, ownerID))
}

func titleAAD(noteID, ownerID int64) string {
	return "note_title:" + itoa(noteID) + ":owner:" + itoa(ownerID)
}

func bodyAAD(noteID, ownerID int64) string {
	return "note_body:" + itoa(noteID) + ":owner:" + itoa(ownerID)
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		buffer[position] = '-'
	}
	return string(buffer[position:])
}
