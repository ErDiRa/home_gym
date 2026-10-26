package maxes

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestDelete(t *testing.T) {
	user, id := uuid.New(), uuid.New()
	f := &fakeQueries{live: map[uuid.UUID]uuid.UUID{id: user}}
	svc := &Service{q: f}
	if err := svc.Delete(context.Background(), user, id); err != nil {
		t.Fatal(err)
	}
	// Gone now, and another user's entry looks missing.
	wantError(t, svc.Delete(context.Background(), user, id), service.KindNotFound, "1RM not found")
	other := uuid.New()
	f.live[other] = uuid.New()
	wantError(t, svc.Delete(context.Background(), user, other), service.KindNotFound, "1RM not found")
	if len(f.deleted) != 1 {
		t.Errorf("deleted %d, want 1", len(f.deleted))
	}
}
