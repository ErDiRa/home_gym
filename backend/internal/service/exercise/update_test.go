package exercise

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
	if _, err := (&Service{q: f}).Update(context.Background(), id, validInput()); err != nil {
		t.Fatal(err)
	}
	if f.updated[0].ID != id {
		t.Errorf("updated %v, want %v", f.updated[0].ID, id)
	}

	f = &fakeQueries{err: pgx.ErrNoRows}
	_, err := (&Service{q: f}).Update(context.Background(), id, validInput())
	wantError(t, err, service.KindNotFound, "exercise not found")

	_, err = (&Service{q: f}).Update(context.Background(), id, Input{})
	wantError(t, err, service.KindInvalid, "name, joint_ids and muscle_group_ids are required")
}
