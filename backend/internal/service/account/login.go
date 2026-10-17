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

// LoginUser checks the credentials against the users table only, for the client app.
// An admin's credentials fail like wrong ones, so the response doesn't reveal the account.
func (s *Service) LoginUser(ctx context.Context, email, password string) (Account, error) {
	return login(email, password, func(email string) (Account, string, error) {
		user, err := s.q.GetUserByEmail(ctx, email)
		return Account{ID: user.ID, Email: user.Email}, user.PasswordHash, err
	})
}

// LoginAdmin checks the credentials against the admins table only, for the admin app.
// A user's credentials fail like wrong ones, so the response doesn't reveal the account.
func (s *Service) LoginAdmin(ctx context.Context, email, password string) (Account, error) {
	return login(email, password, func(email string) (Account, string, error) {
		admin, err := s.q.GetAdminByEmail(ctx, email)
		return Account{ID: admin.ID, Email: admin.Email, IsAdmin: true}, admin.PasswordHash, err
	})
}

// login looks the email up with find, which returns the account and its password hash
// (pgx.ErrNoRows when there is none), and checks the password against it.
func login(email, password string, find func(email string) (Account, string, error)) (Account, error) {
	// Emails are stored lower-cased, so normalize before the lookup.
	account, hash, err := find(strings.ToLower(strings.TrimSpace(email)))
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
