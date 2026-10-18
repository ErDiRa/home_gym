package account

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
)

// dummyHash lets a login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

// Login checks the credentials. An unknown email fails like a wrong password, so the response
// doesn't reveal which accounts exist.
func (s *Service) Login(ctx context.Context, email, password string) (Account, error) {
	// Emails are stored lower-cased, so normalize before the lookup.
	user, err := s.q.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		// Burn a bcrypt comparison anyway, so response timing doesn't reveal which emails exist.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Account{}, service.Unauthorized("invalid email or password")
	}
	if err != nil {
		return Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return Account{}, service.Unauthorized("invalid email or password")
	}
	return Account{ID: user.ID, Email: user.Email}, nil
}
