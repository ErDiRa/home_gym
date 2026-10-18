package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account/repos"
)

func newFake() (*fakeQueries, uuid.UUID) {
	userID := uuid.New()
	return &fakeQueries{
		users: map[string]repos.GetUserByEmailRow{"me@example.com": {ID: userID, Email: "me@example.com", PasswordHash: hash("my-pass")}},
	}, userID
}

func TestLogin(t *testing.T) {
	f, userID := newFake()
	s := &Service{q: f}

	user, err := s.Login(context.Background(), " Me@Example.com ", "my-pass")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != userID || user.Email != "me@example.com" {
		t.Errorf("user = %+v", user)
	}
}

func TestLoginRejects(t *testing.T) {
	f, _ := newFake()
	s := &Service{q: f}

	for _, c := range []struct{ name, email, password string }{
		{"wrong password", "me@example.com", "wrong"},
		{"unknown email", "nobody@example.com", "my-pass"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.Login(context.Background(), c.email, c.password)
			wantError(t, err, service.KindUnauthorized, "invalid email or password")
		})
	}

	// A failing database is not a wrong password.
	f.err = errors.New("connection refused")
	_, err := s.Login(context.Background(), "me@example.com", "my-pass")
	var svcErr *service.Error
	if err == nil || errors.As(err, &svcErr) {
		t.Errorf("err = %v, want the database error", err)
	}
}
