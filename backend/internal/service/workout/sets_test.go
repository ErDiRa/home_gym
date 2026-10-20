package workout

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func TestLogSet(t *testing.T) {
	f, reps, rpe := newFake()
	f.workout = &repos.Workout{ID: uuid.New()}
	userID := uuid.New()
	in := SetInput{ExerciseID: uuid.New(), BlockPosition: 2, ExercisePosition: 1, SetNumber: 3,
		RepsValue: ptr(8.0), RepsUnit: &reps, LoadValue: ptr(8.5), LoadUnit: &rpe}

	if _, err := (&Service{q: f}).LogSet(context.Background(), userID, f.workout.ID, in); err != nil {
		t.Fatal(err)
	}
	u := f.upserts[0]
	if u.UserID != userID || u.WorkoutID != f.workout.ID || u.BlockPosition != 2 || u.SetNumber != 3 || *u.RepsValue != 8 || *u.LoadValue != 8.5 {
		t.Errorf("upserted %+v", u)
	}
}

// A set logged partly keeps what was entered: here only the load, no reps.
func TestLogSetLoadOnly(t *testing.T) {
	f, reps, rpe := newFake()
	f.workout = &repos.Workout{ID: uuid.New()}
	in := SetInput{ExerciseID: uuid.New(), BlockPosition: 1, ExercisePosition: 1, SetNumber: 1,
		RepsUnit: &reps, LoadValue: ptr(7.0), LoadUnit: &rpe}

	if _, err := (&Service{q: f}).LogSet(context.Background(), uuid.New(), f.workout.ID, in); err != nil {
		t.Fatal(err)
	}
	if u := f.upserts[0]; u.RepsValue != nil || *u.LoadValue != 7 {
		t.Errorf("upserted %+v", u)
	}
}

func TestLogSetRejects(t *testing.T) {
	_, reps, rpe := newFake()
	valid := func() SetInput {
		return SetInput{ExerciseID: uuid.New(), BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: ptr(5.0), RepsUnit: &reps}
	}
	tests := []struct {
		name    string
		edit    func(*SetInput)
		wantMsg string
	}{
		{"no exercise", func(in *SetInput) { in.ExerciseID = uuid.Nil }, "exercise_id is required"},
		{"no block", func(in *SetInput) { in.BlockPosition = 0 }, "block_position and exercise_position must be 1 or more"},
		{"set 0", func(in *SetInput) { in.SetNumber = 0 }, "set_number must be 1-100"},
		{"no reps unit", func(in *SetInput) { in.RepsUnit = nil }, "reps_unit is required"},
		{"nothing entered", func(in *SetInput) { in.RepsValue = nil }, "enter the reps or the load"},
		{"load without unit", func(in *SetInput) { in.LoadValue = ptr(10.0) }, "load needs a value and a unit"},
		{"unknown reps unit", func(in *SetInput) { in.RepsUnit = ptr(uuid.New()) }, "unknown reps unit"},
		{"reps below unit", func(in *SetInput) { in.RepsValue = ptr(0.0) }, "reps in reps must be between 1 and 99999"},
		{"load above unit", func(in *SetInput) { in.LoadValue, in.LoadUnit = ptr(11.0), &rpe }, "load in RPE must be between 1 and 10"},
		{"unknown load unit", func(in *SetInput) { in.LoadValue, in.LoadUnit = ptr(11.0), ptr(uuid.New()) }, "unknown load unit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			f.repUnits[0].ID, f.loadUnits[0].ID = reps, rpe
			f.workout = &repos.Workout{ID: uuid.New()}
			in := valid()
			tt.edit(&in)
			_, err := (&Service{q: f}).LogSet(context.Background(), uuid.New(), f.workout.ID, in)
			wantError(t, err, service.KindInvalid, tt.wantMsg)
			if len(f.upserts) != 0 {
				t.Error("logged the set anyway")
			}
		})
	}

	t.Run("another user's workout", func(t *testing.T) {
		f, reps, _ := newFake()
		_, err := (&Service{q: f}).LogSet(context.Background(), uuid.New(), uuid.New(),
			SetInput{ExerciseID: uuid.New(), BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: ptr(5.0), RepsUnit: &reps})
		wantError(t, err, service.KindNotFound, "workout not found")
	})

	t.Run("another user's exercise", func(t *testing.T) {
		f, reps, _ := newFake()
		f.workout = &repos.Workout{ID: uuid.New()}
		f.foreignExer = true
		_, err := (&Service{q: f}).LogSet(context.Background(), uuid.New(), f.workout.ID,
			SetInput{ExerciseID: uuid.New(), BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: ptr(5.0), RepsUnit: &reps})
		wantError(t, err, service.KindInvalid, "unknown exercise")
	})
}

func TestDeleteSet(t *testing.T) {
	f, _, _ := newFake()
	f.workout = &repos.Workout{ID: uuid.New()}
	f.deleteRows = 1
	if err := (&Service{q: f}).DeleteSet(context.Background(), uuid.New(), f.workout.ID, uuid.New()); err != nil {
		t.Fatal(err)
	}

	f.deleteRows = 0
	err := (&Service{q: f}).DeleteSet(context.Background(), uuid.New(), f.workout.ID, uuid.New())
	wantError(t, err, service.KindNotFound, "set not found")

	f.workout = nil
	err = (&Service{q: f}).DeleteSet(context.Background(), uuid.New(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "workout not found")
}
