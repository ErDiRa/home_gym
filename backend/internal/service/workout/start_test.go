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
	unit := repos.GetUnitRow{ID: uuid.New(), Name: "Day A", TrainingID: uuid.New()}
	f.unit = &unit
	userID := uuid.New()

	got, err := (&Service{q: f}).Start(context.Background(), userID, unit.ID)
	if err != nil {
		t.Fatal(err)
	}
	c := f.created[0]
	if c.UserID != userID || *c.UnitID != unit.ID || *c.TrainingID != unit.TrainingID || c.UnitName != "Day A" {
		t.Errorf("created %+v", c)
	}
	// A new workout has no sets yet: [], not null.
	body, _ := json.Marshal(got)
	if got.UnitName != "Day A" || !strings.Contains(string(body), `"sets":[]`) {
		t.Errorf("workout = %s", body)
	}
}

func TestStartRejects(t *testing.T) {
	f, _, _ := newFake() // no unit: unknown or another user's
	_, err := (&Service{q: f}).Start(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindInvalid, "unknown unit")

	f.unit = &repos.GetUnitRow{ID: uuid.New(), Name: "Day A", TrainingID: uuid.New()}
	f.createErr = &pgconn.PgError{Code: "23505"} // one is in progress
	_, err = (&Service{q: f}).Start(context.Background(), uuid.New(), f.unit.ID)
	wantError(t, err, service.KindConflict, "finish or discard the workout in progress first")
}
