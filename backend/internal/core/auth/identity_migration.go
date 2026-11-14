package auth

import (
	"context"
	"github.com/danii1kq/srunal/internal/platform/secure"
)

// Earlier accounts used the content key for identity; migrate ciphertext before starting delivery.
func (s *Service) MigrateIdentityKeys(ctx context.Context) error {
	if s.content.ActiveKeyID() == s.identity.ActiveKeyID() {
		return nil
	}
	for {
		rows, err := s.pool.Query(ctx, `SELECT id,telegram_id_sealed,key_id FROM users WHERE key_id=$1 ORDER BY id LIMIT 100`, s.content.ActiveKeyID())
		if err != nil {
			return err
		}
		type entry struct {
			id     int64
			sealed []byte
			key    string
		}
		batch := []entry{}
		for rows.Next() {
			var e entry
			if err = rows.Scan(&e.id, &e.sealed, &e.key); err != nil {
				rows.Close()
				return err
			}
			batch = append(batch, e)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		for _, e := range batch {
			plain, err := s.content.Open(secure.Sealed{Ciphertext: e.sealed, KeyID: e.key}, "user:telegram_id")
			if err != nil {
				return err
			}
			sealed, err := s.identity.Seal(plain, "user:telegram_id")
			if err != nil {
				return err
			}
			if _, err = s.pool.Exec(ctx, `UPDATE users SET telegram_id_sealed=$2,key_id=$3 WHERE id=$1 AND key_id=$4 AND telegram_id_sealed=$5`, e.id, sealed.Ciphertext, sealed.KeyID, e.key, e.sealed); err != nil {
				return err
			}
		}
	}
}
