package client

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestCreate(t *testing.T) {
	f := &fakeQueries{}
	got, err := newTestService(f).Create(context.Background(), Input{
		Email: "  Anna@Example.COM ", FirstName: " Anna ", LastName: "Muster", Password: "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "anna@example.com" || got.FirstName != "Anna" {
		t.Errorf("created = %+v, want a normalized email and trimmed names", got)
	}
	if bcrypt.CompareHashAndPassword([]byte(f.created[0].PasswordHash), []byte("secret123")) != nil {
		t.Error("the stored hash doesn't match the password")
	}
}

func TestCreateRejects(t *testing.T) {
	valid := Input{Email: "anna@example.com", FirstName: "Anna", LastName: "Muster", Password: "secret123"}
	tests := []struct {
		name     string
		edit     func(*Input)
		fake     fakeQueries
		wantKind service.Kind
		wantMsg  string
	}{
		{"invalid email", func(in *Input) { in.Email = "anna" }, fakeQueries{}, service.KindInvalid, "a valid email is required"},
		{"display name form", func(in *Input) { in.Email = "Anna <anna@example.com>" }, fakeQueries{}, service.KindInvalid, "a valid email is required"},
		{"missing last name", func(in *Input) { in.LastName = " " }, fakeQueries{}, service.KindInvalid, "first_name and last_name are required (max 100 characters)"},
		{"long first name", func(in *Input) { in.FirstName = strings.Repeat("a", 101) }, fakeQueries{}, service.KindInvalid, "first_name and last_name are required (max 100 characters)"},
		{"short password", func(in *Input) { in.Password = "short" }, fakeQueries{}, service.KindInvalid, "password must be 8 to 72 characters"},
		{"long password", func(in *Input) { in.Password = strings.Repeat("ü", 37) }, fakeQueries{}, service.KindInvalid, "password must be 8 to 72 characters"},
		{"admin email", func(*Input) {}, fakeQueries{adminEmails: map[string]bool{"anna@example.com": true}}, service.KindConflict, "this email is already in use"},
		{"client email", func(*Input) {}, fakeQueries{writeErr: &pgconn.PgError{Code: "23505"}}, service.KindConflict, "this email is already in use"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.edit(&in)
			f := tt.fake
			_, err := newTestService(&f).Create(context.Background(), in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
			if len(f.created) != 0 {
				t.Error("created a client anyway")
			}
		})
	}
}
