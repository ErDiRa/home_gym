package training

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestGetAssemblesTree(t *testing.T) {
	f, _, _ := newFake()
	id, dayA, dayB, block1, block2 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	squat, row := uuid.New(), uuid.New()
	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	f.training = repos.GetTrainingRow{ID: id, Name: "Strength"}
	f.units = []repos.ListTrainingUnitsRow{
		{ID: dayA, Name: "Day A", Weekdays: []int16{1}, Dates: []pgtype.Date{{Time: date, Valid: true}}},
		{ID: dayB, Name: "Day B"}, // no weekdays, dates or blocks
	}
	f.blocks = []repos.ListTrainingBlocksRow{
		{ID: block1, UnitID: dayA, Kind: "single"},
		{ID: block2, UnitID: dayA, Kind: "superset", Sets: ptr[int32](3)},
	}
	f.exercises = []repos.ListTrainingExercisesRow{
		{BlockID: block1, ExerciseID: squat, Name: "Squat", RepsValue: ptr(5.0), RepsUnitName: "reps"},
		{BlockID: block1, ExerciseID: row, Name: "Row"},
	}

	got, err := newTestService(f).Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Units) != 2 || got.Units[0].ID != dayA || got.Units[1].ID != dayB {
		t.Fatalf("units = %+v", got.Units)
	}
	a := got.Units[0]
	if len(a.Dates) != 1 || a.Dates[0] != "2026-03-01" {
		t.Errorf("dates = %v", a.Dates)
	}
	if len(a.Blocks) != 2 || a.Blocks[0].ID != block1 || a.Blocks[1].ID != block2 {
		t.Fatalf("blocks = %+v", a.Blocks)
	}
	ex := a.Blocks[0].Exercises
	if len(ex) != 2 || ex[0].ExerciseID != squat || ex[1].ExerciseID != row || *ex[0].RepsValue != 5 || ex[0].RepsUnitName != "reps" {
		t.Errorf("exercises = %+v", ex)
	}

	// Empty lists encode as [], not null, and the row's fields sit at the top level.
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"name":"Strength"`, `"weekdays":[],"dates":[],"blocks":[]`, `"exercises":[]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("JSON %s lacks %s", body, want)
		}
	}
}

func TestGetMissing(t *testing.T) {
	f, _, _ := newFake()
	f.getErr = pgx.ErrNoRows

	_, err := newTestService(f).Get(context.Background(), uuid.New())
	wantError(t, err, service.KindNotFound, "training not found")
}
