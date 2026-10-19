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

	got, err := (&Service{q: f}).Start(context.Background(), userID, slot.ID)
	if err != nil {
		t.Fatal(err)
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
	_, err := (&Service{q: f}).Start(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindInvalid, "unknown planned session")

	f.slot = &repos.GetWeekSessionRow{ID: uuid.New(), TrainingID: uuid.New(), SessionID: uuid.New(), Name: "Session 1"}
	f.createErr = &pgconn.PgError{Code: "23505"} // one is in progress
	_, err = (&Service{q: f}).Start(context.Background(), uuid.New(), f.slot.ID)
	wantError(t, err, service.KindConflict, "finish or discard the workout in progress first")
}
