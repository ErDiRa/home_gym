package app

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/common/errs"
)

// dummyHash lets a login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)

// Login checks the credentials and starts a new session; the login time becomes the session's
// AuthTime. An unknown email fails like a wrong password, so the response doesn't reveal which
// accounts exist.
func (s *Service) Login(ctx context.Context, email, password string, remember bool) (Session, error) {
	// Emails are stored lower-cased, so normalize before the lookup.
	user, err := s.users.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, ErrUserNotFound) {
		// Burn a bcrypt comparison anyway, so response timing doesn't reveal which emails exist.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Session{}, errs.Unauthorized("invalid email or password")
	}
	if err != nil {
		return Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return Session{}, errs.Unauthorized("invalid email or password")
	}
	return s.issue(user.Account, s.now(), remember)
}
