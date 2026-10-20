package workout

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func TestStart(t *testing.T) {
	f, _, _ := newFake()
	slot := repos.GetWeekSessionRow{ID: uuid.New(), TrainingID: uuid.New(), SessionID: uuid.New(), Name: "Session 1"}
	f.slot = &slot
	userID := uuid.New()

	got, created, err := (&Service{q: f}).Start(context.Background(), userID, slot.ID)
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}
	c := f.created[0]
	if c.UserID != userID || *c.WeekSessionID != slot.ID || *c.SessionID != slot.SessionID || *c.TrainingID != slot.TrainingID || c.SessionName != "Session 1" {
		t.Errorf("created %+v", c)
	}
	// A new workout has no sets yet: [], not null.
	body, _ := json.Marshal(got)
	if got.SessionName != "Session 1" || *got.WeekSessionID != slot.ID || !strings.Contains(string(body), `"sets":[]`) {
		t.Errorf("workout = %s", body)
	}
}

func TestStartRejects(t *testing.T) {
	f, _, _ := newFake() // no slot: unknown or of another user's training
	_, _, err := (&Service{q: f}).Start(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindInvalid, "unknown planned session")
}

// Starting a slot that is already in progress returns that workout, with its sets.
func TestStartSlotInProgress(t *testing.T) {
	f, _, _ := newFake()
	f.slot = &repos.GetWeekSessionRow{ID: uuid.New(), TrainingID: uuid.New(), SessionID: uuid.New(), Name: "Session 1"}
	f.createErr = &pgconn.PgError{Code: "23505"} // idx_workouts_slot_in_progress
	existing := repos.Workout{ID: uuid.New(), WeekSessionID: &f.slot.ID, SessionName: "Session 1"}
	f.inSlot = &existing
	f.sets = []repos.ListWorkoutSetsRow{{ID: uuid.New(), SetNumber: 1}}

	got, created, err := (&Service{q: f}).Start(context.Background(), uuid.New(), f.slot.ID)
	if err != nil || created || got.ID != existing.ID || len(got.Sets) != 1 {
		t.Errorf("got %+v, created %v, err %v", got, created, err)
	}
}
