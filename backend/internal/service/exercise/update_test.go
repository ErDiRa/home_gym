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
	userID, id := uuid.New(), uuid.New()
	if _, err := (&Service{q: f}).Update(context.Background(), userID, id, validInput()); err != nil {
		t.Fatal(err)
	}
	if f.updated[0].ID != id || f.updated[0].UserID != userID {
		t.Errorf("updated %v of %v, want %v of %v", f.updated[0].ID, f.updated[0].UserID, id, userID)
	}

	f = &fakeQueries{err: pgx.ErrNoRows}
	_, err := (&Service{q: f}).Update(context.Background(), userID, id, validInput())
	wantError(t, err, service.KindNotFound, "exercise not found")

	_, err = (&Service{q: f}).Update(context.Background(), userID, id, Input{})
	wantError(t, err, service.KindInvalid, "name, joint_ids and muscle_group_ids are required")
}
