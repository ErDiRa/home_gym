package account

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
)

// Current returns the signed-in user from their token's id. A user deleted since login is
// NotFound, so the session can be ended.
func (s *Service) Current(ctx context.Context, id uuid.UUID) (Account, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, service.NotFound("account no longer exists", err)
	}
	if err != nil {
		return Account{}, err
	}
	return Account{ID: user.ID, Email: user.Email}, nil
}
