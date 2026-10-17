package training

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestDuplicate(t *testing.T) {
	f, _, _ := newFake()
	id, unitID, blockID, exerciseID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.training = repos.GetTrainingRow{ID: id, ClientID: uuid.New(), Name: "Strength", Description: "Base", Active: true,
		StartDate: pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		EndDate:   pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true}}
	f.units = []repos.ListTrainingUnitsRow{{ID: unitID, Name: "Day A", Weekdays: []int16{2},
		Dates: []pgtype.Date{{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}}}}
	f.blocks = []repos.ListTrainingBlocksRow{{ID: blockID, UnitID: unitID, Kind: "superset", Sets: ptr[int32](3)}}
	f.exercises = []repos.ListTrainingExercisesRow{{BlockID: blockID, ExerciseID: exerciseID, Tempo: "31X1"}}
	client := uuid.New()

	got, err := newTestService(f).Duplicate(context.Background(), id, client)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == id || got.ClientID != client || got.Name != "Strength" || got.Description != "Base" {
		t.Errorf("copy = %+v", got)
	}
	// The copy is the next 4-week phase: it starts the day after the source ends.
	if start, end := got.StartDate.Time.Format(time.DateOnly), got.EndDate.Time.Format(time.DateOnly); start != "2026-03-29" || end != "2026-04-25" {
		t.Errorf("period = %s to %s, want 2026-03-29 to 2026-04-25", start, end)
	}
	if got.Active || f.called("DeactivateClientTrainings") {
		t.Error("the copy of an active training is active too")
	}
	if len(f.insertedUnits) != 1 || f.insertedUnits[0].TrainingID != got.ID || len(f.insertedUnits[0].Dates) != 1 || f.insertedUnits[0].Weekdays[0] != 2 {
		t.Errorf("units = %+v", f.insertedUnits)
	} else if d := f.insertedUnits[0].Dates[0].Time.Format(time.DateOnly); d != "2026-03-29" {
		t.Errorf("unit date = %s, want it moved with the period to 2026-03-29", d)
	}
	if len(f.insertedBlocks) != 1 || f.insertedBlocks[0].Kind != "superset" || *f.insertedBlocks[0].Sets != 3 {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
	if len(f.insertedExercises) != 1 || f.insertedExercises[0].ExerciseID != exerciseID || f.insertedExercises[0].Tempo != "31X1" {
		t.Errorf("exercises = %+v", f.insertedExercises)
	}
}

func TestDuplicateErrors(t *testing.T) {
	f, _, _ := newFake()
	f.getErr = pgx.ErrNoRows
	_, err := newTestService(f).Duplicate(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "training not found")

	f, _, _ = newFake()
	f.clientExists = false
	_, err = newTestService(f).Duplicate(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindInvalid, "unknown client")
}
