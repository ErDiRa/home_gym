// Package app holds the identity module's business logic: signing in and the rules of a session.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Account is a signed-in user as the frontend sees it.
type Account struct {
	ID    uuid.UUID
	Email string
}

// User is an account with what logging in checks.
type User struct {
	Account
	PasswordHash string // bcrypt
}

// ErrUserNotFound is what a UserRepository returns for an unknown (or deleted) user.
var ErrUserNotFound = errors.New("user not found")

// UserRepository reads the users; they are created by hand, there is no sign-up.
type UserRepository interface {
	// UserByEmail returns the live user with the (lower-cased) email, or ErrUserNotFound.
	UserByEmail(ctx context.Context, email string) (User, error)
	// AccountByID returns the live user with the id, or ErrUserNotFound.
	AccountByID(ctx context.Context, id uuid.UUID) (Account, error)
}

// Claims are what a session token says.
type Claims struct {
	UserID   uuid.UUID
	AuthTime time.Time // the original login, from which MaxSessionAge is measured
	Remember bool      // keep the cookie when the browser closes
}

// Tokens signs and checks session tokens.
type Tokens interface {
	Issue(claims Claims, expiry time.Time) (string, error)
	// Parse returns the claims of a valid, unexpired token, or an error.
	Parse(token string) (Claims, error)
}

// Service implements the identity usecases.
type Service struct {
	users  UserRepository
	tokens Tokens
	now    func() time.Time
}

func NewService(users UserRepository, tokens Tokens) *Service {
	if users == nil {
		panic("users cannot be nil")
	}
	if tokens == nil {
		panic("tokens cannot be nil")
	}
	return &Service{users: users, tokens: tokens, now: time.Now}
}
