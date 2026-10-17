package lookup

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestCreateFocus(t *testing.T) {
	f := &fakeQueries{}
	got, err := (&Service{q: f}).CreateFocus(context.Background(), FocusInput{Name: " Hüftmobilität ", Alias: " HÜFTE "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Hüftmobilität" || got.Alias != "HÜFTE" {
		t.Errorf("focus = %+v, want trimmed name and alias", got)
	}
}

func TestCreateFocusRejects(t *testing.T) {
	tests := []struct {
		name     string
		in       FocusInput
		err      error
		wantKind service.Kind
		wantMsg  string
	}{
		{"no name", FocusInput{Alias: "HIP"}, nil, service.KindInvalid, "name is required (max 60 characters)"},
		{"long name", FocusInput{Name: strings.Repeat("x", 61), Alias: "HIP"}, nil, service.KindInvalid, "name is required (max 60 characters)"},
		{"no alias", FocusInput{Name: "Hip"}, nil, service.KindInvalid, "alias is required (max 10 characters)"},
		{"long alias", FocusInput{Name: "Hip", Alias: strings.Repeat("x", 11)}, nil, service.KindInvalid, "alias is required (max 10 characters)"},
		{"duplicate", FocusInput{Name: "Hip", Alias: "HIP"}, &pgconn.PgError{Code: "23505"}, service.KindConflict, "this focus already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&Service{q: &fakeQueries{err: tt.err}}).CreateFocus(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
		})
	}
}
