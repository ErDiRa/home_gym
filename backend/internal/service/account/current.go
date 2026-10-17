package account

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
)

// Current returns the signed-in account from its token's role and id. An account deleted
// since login is NotFound, so the session can be ended.
func (s *Service) Current(ctx context.Context, role string, id uuid.UUID) (Account, error) {
	var (
		account Account
		err     error
	)
	if role == auth.RoleAdmin {
		admin, e := s.q.GetAdminByID(ctx, id)
		account, err = Account{ID: admin.ID, Email: admin.Email, IsAdmin: true}, e
	} else {
		user, e := s.q.GetUserByID(ctx, id)
		account, err = Account{ID: user.ID, Email: user.Email}, e
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, service.NotFound("account no longer exists", err)
	}
	if err != nil {
		return Account{}, err
	}
	return account, nil
}
