package account

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
)

// dummyHash lets Login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

// Login checks the credentials and returns the account they belong to.
func (s *Service) Login(ctx context.Context, email, password string) (Account, error) {
	// Emails are stored lower-cased, so normalize before the lookup.
	email = strings.ToLower(strings.TrimSpace(email))

	account, hash, err := s.findAccount(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		// Burn a bcrypt comparison anyway, so response timing doesn't reveal which emails exist.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Account{}, service.Unauthorized("invalid email or password")
	}
	if err != nil {
		return Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Account{}, service.Unauthorized("invalid email or password")
	}
	return account, nil
}

// findAccount looks the email up in the users table, then in the admins table, returning the
// account and its password hash. It returns pgx.ErrNoRows when neither has it.
func (s *Service) findAccount(ctx context.Context, email string) (Account, string, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err == nil {
		return Account{ID: user.ID, Email: user.Email}, user.PasswordHash, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, "", err
	}
	admin, err := s.q.GetAdminByEmail(ctx, email)
	if err != nil {
		return Account{}, "", err
	}
	return Account{ID: admin.ID, Email: admin.Email, IsAdmin: true}, admin.PasswordHash, nil
}
