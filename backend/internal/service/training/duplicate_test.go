package training

import (
	"context"
	"slices"
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
	id, s1, s2, blockID, exerciseID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength", Description: "Base", Active: true, Weeks: 2,
		StartDate: pgtype.Date{Time: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), Valid: true},
		EndDate:   pgtype.Date{Time: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), Valid: true}}
	f.sessions = []repos.ListTrainingSessionsRow{{ID: s1, Name: "Session 1", Notes: "Heavy"}, {ID: s2, Name: "Session 2"}}
	f.blocks = []repos.ListTrainingBlocksRow{{ID: blockID, SessionID: s1, Kind: "superset", Sets: ptr[int32](3)}}
	f.exercises = []repos.ListTrainingExercisesRow{{BlockID: blockID, ExerciseID: exerciseID, Tempo: "31X1"}}
	f.slots = []repos.ListWeekSessionsRow{
		{ID: uuid.New(), Week: 1, Position: 1, SessionID: s2},
		{ID: uuid.New(), Week: 1, Position: 2, SessionID: s1},
		{ID: uuid.New(), Week: 2, Position: 1, SessionID: s1},
	}
	userID := uuid.New()

	got, err := newTestService(f).Duplicate(context.Background(), userID, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == id || f.created[0].UserID != userID || got.Name != "Strength" || got.Description != "Base" || got.Weeks != 2 {
		t.Errorf("copy = %+v", got)
	}
	if f.gotTraining[0].UserID != userID {
		t.Errorf("copied a training of %v, want one of %v", f.gotTraining[0].UserID, userID)
	}
	// The copy is the next phase: as many weeks, from the Monday after the source ends.
	if start, end := got.StartDate.Time.Format(time.DateOnly), got.EndDate.Time.Format(time.DateOnly); start != "2026-03-16" || end != "2026-03-29" {
		t.Errorf("period = %s to %s, want 2026-03-16 to 2026-03-29", start, end)
	}
	if got.Active || f.created[0].Active {
		t.Error("the copy of an active training is active too")
	}
	if len(f.insertedSessions) != 2 || *f.insertedSessions[0].TrainingID != got.ID || f.insertedSessions[0].Notes != "Heavy" {
		t.Errorf("sessions = %+v", f.insertedSessions)
	}
	if len(f.insertedBlocks) != 1 || f.insertedBlocks[0].Kind != "superset" || *f.insertedBlocks[0].Sets != 3 {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
	if len(f.insertedExercises) != 1 || f.insertedExercises[0].ExerciseID != exerciseID || f.insertedExercises[0].Tempo != "31X1" || f.insertedExercises[0].UserID != userID {
		t.Errorf("exercises = %+v", f.insertedExercises)
	}
	// The schedule is copied onto the new sessions.
	c1, c2 := f.sessionIDs[0], f.sessionIDs[1]
	want := []repos.InsertWeekSessionParams{
		{TrainingID: got.ID, Week: 1, Position: 1, SessionID: c2},
		{TrainingID: got.ID, Week: 1, Position: 2, SessionID: c1},
		{TrainingID: got.ID, Week: 2, Position: 1, SessionID: c1},
	}
	if !slices.Equal(f.insertedSlots, want) {
		t.Errorf("slots = %+v, want %+v", f.insertedSlots, want)
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
