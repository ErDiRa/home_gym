package training

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestGetAssemblesTree(t *testing.T) {
	f, _, _ := newFake()
	id, s1, s2, block1, block2, slot1, slot2 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	squat, row := uuid.New(), uuid.New()
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength", Weeks: 2}
	f.sessions = []repos.ListTrainingSessionsRow{
		{ID: s1, Name: "Session 1"},
		{ID: s2, Name: "Session 2"}, // no blocks
	}
	f.blocks = []repos.ListTrainingBlocksRow{
		{ID: block1, SessionID: s1, Kind: "single"},
		{ID: block2, SessionID: s1, Kind: "superset", Sets: ptr[int32](3)},
	}
	f.exercises = []repos.ListTrainingExercisesRow{
		{BlockID: block1, ExerciseID: squat, Name: "Squat", RepsValue: ptr(5.0), RepsUnitName: "reps"},
		{BlockID: block1, ExerciseID: row, Name: "Row"},
	}
	f.slots = []repos.ListWeekSessionsRow{
		{ID: slot1, Week: 1, Position: 1, SessionID: s1},
		{ID: slot2, Week: 1, Position: 2, SessionID: s1},
	}

	got, err := newTestService(f).Get(context.Background(), uuid.New(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sessions) != 2 || got.Sessions[0].ID != s1 || got.Sessions[1].ID != s2 {
		t.Fatalf("sessions = %+v", got.Sessions)
	}
	a := got.Sessions[0]
	if len(a.Blocks) != 2 || a.Blocks[0].ID != block1 || a.Blocks[1].ID != block2 {
		t.Fatalf("blocks = %+v", a.Blocks)
	}
	ex := a.Blocks[0].Exercises
	if len(ex) != 2 || ex[0].ExerciseID != squat || ex[1].ExerciseID != row || *ex[0].RepsValue != 5 || ex[0].RepsUnitName != "reps" {
		t.Errorf("exercises = %+v", ex)
	}
	// One list per week, the empty week included.
	if len(got.Schedule) != 2 || len(got.Schedule[0]) != 2 || got.Schedule[0][1] != (Slot{ID: slot2, SessionID: s1}) || len(got.Schedule[1]) != 0 {
		t.Errorf("schedule = %+v", got.Schedule)
	}

	// Empty lists encode as [], not null, and the row's fields sit at the top level.
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"name":"Strength"`, `"name":"Session 2","notes":"","blocks":[]`, `"exercises":[]`, `"schedule":[[{"id":`, `],[]]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("JSON %s lacks %s", body, want)
		}
	}
}

func TestGetMissing(t *testing.T) {
	f, _, _ := newFake()
	f.getErr = pgx.ErrNoRows

	_, err := newTestService(f).Get(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "training not found")
}
