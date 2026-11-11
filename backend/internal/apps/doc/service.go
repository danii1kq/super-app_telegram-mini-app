package doc

import (
	"context"

	"github.com/danii1kq/srunal/internal/platform/postgres"
)

// Service resolves the SruNal account to its DocGen account and asks the client
// for data. The link is the only thing SruNal stores about SruDoc.
type Service struct {
	pool   *postgres.Pool
	client *Client
}

func NewService(pool *postgres.Pool, client *Client) *Service {
	return &Service{pool: pool, client: client}
}

// externalID returns the confirmed DocGen account id for a SruNal user.
func (s *Service) externalID(ctx context.Context, userID int64) (string, error) {
	var externalID string
	err := s.pool.QueryRow(ctx,
		`SELECT external_id FROM app_links WHERE user_id = $1 AND app_id = 'doc'`, userID,
	).Scan(&externalID)
	if err != nil {
		// No link yet is a normal state, not a failure: the app screen offers
		// to connect the account.
		return "", nil
	}
	return externalID, nil
}

// Orders lists the caller's orders.
func (s *Service) Orders(ctx context.Context, userID int64) ([]Order, error) {
	externalID, err := s.externalID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.client.ListOrders(ctx, externalID)
}

// Order returns one order.
func (s *Service) Order(ctx context.Context, userID int64, number string) (Order, error) {
	externalID, err := s.externalID(ctx, userID)
	if err != nil {
		return Order{}, err
	}
	return s.client.OrderStatus(ctx, externalID, number)
}

// Linked reports whether the account is connected to DocGen.
func (s *Service) Linked(ctx context.Context, userID int64) (bool, error) {
	externalID, err := s.externalID(ctx, userID)
	if err != nil {
		return false, err
	}
	return externalID != "", nil
}

// Link stores a confirmed DocGen account id for the user.
func (s *Service) Link(ctx context.Context, userID int64, externalID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO app_links (user_id, app_id, external_id) VALUES ($1, 'doc', $2)
		 ON CONFLICT (user_id, app_id) DO UPDATE SET external_id = excluded.external_id`,
		userID, externalID,
	)
	return err
}
