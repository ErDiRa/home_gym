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
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength", Description: "Base", Active: true, SessionsPerWeek: 4,
		StartDate: pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		EndDate:   pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true}}
	f.units = []repos.ListTrainingUnitsRow{{ID: unitID, Name: "Day A", Notes: "Heavy"}}
	f.blocks = []repos.ListTrainingBlocksRow{{ID: blockID, UnitID: unitID, Kind: "superset", Sets: ptr[int32](3)}}
	f.exercises = []repos.ListTrainingExercisesRow{{BlockID: blockID, ExerciseID: exerciseID, Tempo: "31X1"}}
	userID := uuid.New()

	got, err := newTestService(f).Duplicate(context.Background(), userID, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == id || f.created[0].UserID != userID || got.Name != "Strength" || got.Description != "Base" || got.SessionsPerWeek != 4 {
		t.Errorf("copy = %+v", got)
	}
	if f.gotTraining[0].UserID != userID {
		t.Errorf("copied a training of %v, want one of %v", f.gotTraining[0].UserID, userID)
	}
	// The copy is the next 4-week phase: it starts the day after the source ends.
	if start, end := got.StartDate.Time.Format(time.DateOnly), got.EndDate.Time.Format(time.DateOnly); start != "2026-03-29" || end != "2026-04-25" {
		t.Errorf("period = %s to %s, want 2026-03-29 to 2026-04-25", start, end)
	}
	if got.Active || f.called("DeactivateUserTrainings") {
		t.Error("the copy of an active training is active too")
	}
	if len(f.insertedUnits) != 1 || f.insertedUnits[0].TrainingID != got.ID || f.insertedUnits[0].Notes != "Heavy" {
		t.Errorf("units = %+v", f.insertedUnits)
	}
	if len(f.insertedBlocks) != 1 || f.insertedBlocks[0].Kind != "superset" || *f.insertedBlocks[0].Sets != 3 {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
	if len(f.insertedExercises) != 1 || f.insertedExercises[0].ExerciseID != exerciseID || f.insertedExercises[0].Tempo != "31X1" || f.insertedExercises[0].UserID != userID {
		t.Errorf("exercises = %+v", f.insertedExercises)
	}
}

func TestDuplicateMissing(t *testing.T) {
	f, _, _ := newFake()
	f.getErr = pgx.ErrNoRows // unknown, deleted or another user's
	_, err := newTestService(f).Duplicate(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "training not found")
	if f.called("CreateTraining") {
		t.Error("created a copy anyway")
	}
}
