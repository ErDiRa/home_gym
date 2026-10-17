package client

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestUpdate(t *testing.T) {
	f := &fakeQueries{}
	id := uuid.New()
	// No password needed: Update never changes it.
	got, err := newTestService(f).Update(context.Background(), id, Input{Email: "BEN@example.com", FirstName: "Ben", LastName: "Beispiel"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || f.updated[0].Email != "ben@example.com" {
		t.Errorf("updated = %+v", f.updated)
	}
}

func TestUpdateErrors(t *testing.T) {
	in := Input{Email: "ben@example.com", FirstName: "Ben", LastName: "Beispiel"}

	f := &fakeQueries{writeErr: pgx.ErrNoRows}
	_, err := newTestService(f).Update(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindNotFound, "client not found")

	f = &fakeQueries{adminEmails: map[string]bool{"ben@example.com": true}}
	_, err = newTestService(f).Update(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindConflict, "this email is already in use")
}
