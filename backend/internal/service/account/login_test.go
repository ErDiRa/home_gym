package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account/repos"
)

func newFake() (*fakeQueries, uuid.UUID, uuid.UUID) {
	userID, adminID := uuid.New(), uuid.New()
	return &fakeQueries{
		users:  map[string]repos.GetUserByEmailRow{"anna@example.com": {ID: userID, Email: "anna@example.com", PasswordHash: hash("anna-pass")}},
		admins: map[string]repos.GetAdminByEmailRow{"admin@example.com": {ID: adminID, Email: "admin@example.com", PasswordHash: hash("admin-pass")}},
	}, userID, adminID
}

func TestLogin(t *testing.T) {
	f, userID, adminID := newFake()
	s := &Service{q: f}

	user, err := s.LoginUser(context.Background(), " Anna@Example.com ", "anna-pass")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != userID || user.IsAdmin || user.Role() != auth.RoleUser {
		t.Errorf("user = %+v", user)
	}

	admin, err := s.LoginAdmin(context.Background(), "admin@example.com", "admin-pass")
	if err != nil {
		t.Fatal(err)
	}
	if admin.ID != adminID || !admin.IsAdmin || admin.Role() != auth.RoleAdmin {
		t.Errorf("admin = %+v", admin)
	}
}

func TestLoginRejects(t *testing.T) {
	f, _, _ := newFake()
	s := &Service{q: f}
	for _, c := range []struct {
		name            string
		login           func(ctx context.Context, email, password string) (Account, error)
		email, password string
	}{
		{"user, wrong password", s.LoginUser, "anna@example.com", "wrong"},
		{"user, unknown email", s.LoginUser, "nobody@example.com", "anna-pass"},
		// Each app's login only knows its own accounts, so the other kind fails like a wrong password.
		{"admin at user login", s.LoginUser, "admin@example.com", "admin-pass"},
		{"admin, wrong password", s.LoginAdmin, "admin@example.com", "anna-pass"},
		{"user at admin login", s.LoginAdmin, "anna@example.com", "anna-pass"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.login(context.Background(), c.email, c.password)
			wantError(t, err, service.KindUnauthorized, "invalid email or password")
		})
	}

	// A failing database is not a wrong password.
	f.err = errors.New("connection refused")
	for _, login := range []func(ctx context.Context, email, password string) (Account, error){s.LoginUser, s.LoginAdmin} {
		_, err := login(context.Background(), "anna@example.com", "anna-pass")
		var svcErr *service.Error
		if err == nil || errors.As(err, &svcErr) {
			t.Errorf("err = %v, want the database error", err)
		}
	}
}
