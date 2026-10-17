package training

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestUpdateKeepsUnitIDs(t *testing.T) {
	f, reps, _ := newFake()
	kept := uuid.New()
	f.knownUnits[kept] = true
	in := twoUnitInput(reps)
	in.Units[1].ID = &kept // Day B exists, Day A is new
	in.Active = true
	id := uuid.New()

	got, err := newTestService(f).Update(context.Background(), id, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || !got.Active || got.StartDate.Time.Format(time.DateOnly) != "2026-03-02" || got.EndDate.Time.Format(time.DateOnly) != "2026-03-29" {
		t.Errorf("updated = %+v", got)
	}
	if !slices.Equal(f.keptUnits, []uuid.UUID{kept}) {
		t.Errorf("kept units = %v, want [%v]", f.keptUnits, kept)
	}
	if len(f.insertedUnits) != 1 || f.insertedUnits[0].Name != "Day A" || f.insertedUnits[0].Position != 1 {
		t.Errorf("inserted units = %+v", f.insertedUnits)
	}
	if len(f.updatedUnits) != 1 || f.updatedUnits[0].ID != kept || f.updatedUnits[0].Position != 2 || f.updatedUnits[0].TrainingID != id {
		t.Errorf("updated units = %+v", f.updatedUnits)
	}
	// Blocks are cleared before they are re-inserted.
	if f.indexOf("DeleteTrainingBlocks") > f.indexOf("InsertTrainingBlock") {
		t.Errorf("calls = %v, want DeleteTrainingBlocks first", f.calls)
	}
	if f.indexOf("DeactivateClientTrainings") > f.indexOf("UpdateTraining") {
		t.Errorf("calls = %v, want DeactivateClientTrainings before UpdateTraining", f.calls)
	}
}

func TestUpdateUnknownUnit(t *testing.T) {
	f, reps, _ := newFake()
	foreign := uuid.New() // not in f.knownUnits: belongs to another training
	in := twoUnitInput(reps)
	in.Units[0].ID = &foreign

	_, err := newTestService(f).Update(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown unit")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}

func TestUpdateMissingTraining(t *testing.T) {
	f, reps, _ := newFake()
	f.updateErr = pgx.ErrNoRows

	_, err := newTestService(f).Update(context.Background(), uuid.New(), twoUnitInput(reps))
	wantError(t, err, service.KindNotFound, "training not found")
}
