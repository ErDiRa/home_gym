package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account/repos"
)

// fakeQueries holds users and admins by email; the lookups return pgx.ErrNoRows for others.
type fakeQueries struct {
	users  map[string]repos.GetUserByEmailRow
	admins map[string]repos.GetAdminByEmailRow
	err    error // returned by every lookup when set
}

// hash is a cheap bcrypt hash of password for fixtures.
func hash(password string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return string(h)
}

func (f *fakeQueries) GetUserByEmail(_ context.Context, email string) (repos.GetUserByEmailRow, error) {
	if f.err != nil {
		return repos.GetUserByEmailRow{}, f.err
	}
	u, ok := f.users[email]
	if !ok {
		return u, pgx.ErrNoRows
	}
	return u, nil
}

func (f *fakeQueries) GetUserByID(_ context.Context, id uuid.UUID) (repos.GetUserByIDRow, error) {
	for _, u := range f.users {
		if u.ID == id {
			return repos.GetUserByIDRow{ID: u.ID, Email: u.Email}, nil
		}
	}
	return repos.GetUserByIDRow{}, pgx.ErrNoRows
}

func (f *fakeQueries) GetAdminByEmail(_ context.Context, email string) (repos.GetAdminByEmailRow, error) {
	if f.err != nil {
		return repos.GetAdminByEmailRow{}, f.err
	}
	a, ok := f.admins[email]
	if !ok {
		return a, pgx.ErrNoRows
	}
	return a, nil
}

func (f *fakeQueries) GetAdminByID(_ context.Context, id uuid.UUID) (repos.GetAdminByIDRow, error) {
	for _, a := range f.admins {
		if a.ID == id {
			return repos.GetAdminByIDRow{ID: a.ID, Email: a.Email}, nil
		}
	}
	return repos.GetAdminByIDRow{}, pgx.ErrNoRows
}

func (f *fakeQueries) IsAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := f.GetAdminByID(ctx, id)
	return err == nil, nil
}

func (f *fakeQueries) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := f.GetUserByID(ctx, id)
	return err == nil, nil
}

// wantError fails the test unless err is a *service.Error of kind with message msg.
func wantError(t *testing.T, err error, kind service.Kind, msg string) {
	t.Helper()
	var svcErr *service.Error
	if !errors.As(err, &svcErr) {
		t.Fatalf("err = %v, want a service error %q", err, msg)
	}
	if svcErr.Kind != kind || svcErr.Msg != msg {
		t.Fatalf("err = {kind %d, %q}, want {kind %d, %q}", svcErr.Kind, svcErr.Msg, kind, msg)
	}
}
