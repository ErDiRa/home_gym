package handlers

import (
	"strings"
	"testing"

	"github.com/erdira/homegym/backend/internal/database"
	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func TestCheckBlock(t *testing.T) {
	reps, rpe := uuid.New(), uuid.New()
	lk := trainingLookups{
		kinds: map[string]database.ListBlockKindsRow{
			"single":   {Code: "single", Name: "Single", MinExercises: 1},
			"pair":     {Code: "pair", Name: "Pair", MinExercises: 2, MaxExercises: ptr[int32](2), SharedSets: true},
			"superset": {Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
			"complex":  {Code: "complex", Name: "Complex", MinExercises: 2, SharedSets: true, SharedReps: true, SharedLoad: true},
		},
		repUnits:  map[uuid.UUID]unit{reps: {name: "reps", min: ptr(1.0)}},
		loadUnits: map[uuid.UUID]unit{rpe: {name: "RPE", min: ptr(1.0), max: ptr(10.0)}},
	}
	withReps := amount{RepsValue: ptr(8.0), RepsUnit: &reps}
	exercise := func(a amount) trainingExerciseRequest {
		return trainingExerciseRequest{ExerciseID: uuid.New(), Sets: ptr[int32](3), amount: a}
	}

	tests := []struct {
		name    string
		block   trainingBlockRequest
		wantErr string // substring; empty means valid
	}{
		{"single", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(withReps)}}, ""},
		{"single with two exercises", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(withReps), exercise(withReps)}}, ""},
		{"single without exercises", trainingBlockRequest{Kind: "single"}, "at least 1 exercise"},
		{"pair with three exercises", trainingBlockRequest{Kind: "pair", Sets: ptr[int32](3), Exercises: []trainingExerciseRequest{exercise(withReps), exercise(withReps), exercise(withReps)}}, "exactly 2"},
		{"superset with one exercise", trainingBlockRequest{Kind: "superset", Sets: ptr[int32](3), Exercises: []trainingExerciseRequest{exercise(withReps)}}, "at least 2"},
		{"unknown kind", trainingBlockRequest{Kind: "circuit", Sets: ptr[int32](3)}, "unknown block kind"},
		{"superset without sets", trainingBlockRequest{Kind: "superset", Exercises: []trainingExerciseRequest{exercise(withReps), exercise(withReps)}}, "sets must be"},
		{"single exercise without sets", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(withReps), {ExerciseID: uuid.New(), amount: withReps}}}, "exercise 2: sets must be"},
		{"exercise without reps", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(amount{})}}, "reps are required"},
		{"complex needs block reps", trainingBlockRequest{Kind: "complex", Sets: ptr[int32](5), Exercises: []trainingExerciseRequest{exercise(amount{}), exercise(amount{})}}, "reps are required"},
		{"complex", trainingBlockRequest{Kind: "complex", Sets: ptr[int32](5), amount: withReps, Exercises: []trainingExerciseRequest{exercise(amount{}), exercise(amount{})}}, ""},
		{"range must rise", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(amount{RepsValue: ptr(10.0), RepsMax: ptr(8.0), RepsUnit: &reps})}}, "higher"},
		{"load out of unit bounds", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(amount{RepsValue: ptr(8.0), RepsUnit: &reps, LoadValue: ptr(12.0), LoadUnit: &rpe})}}, "between 1 and 10"},
		{"load without unit", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(amount{RepsValue: ptr(8.0), RepsUnit: &reps, LoadValue: ptr(80.0)})}}, "value and a unit"},
		{"unknown unit", trainingBlockRequest{Kind: "single", Exercises: []trainingExerciseRequest{exercise(amount{RepsValue: ptr(8.0), RepsUnit: ptr(uuid.New())})}}, "unknown reps unit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkBlock(&tt.block, lk)
			if tt.wantErr == "" && got != "" {
				t.Fatalf("checkBlock() = %q, want no error", got)
			}
			if tt.wantErr != "" && !strings.Contains(got, tt.wantErr) {
				t.Fatalf("checkBlock() = %q, want it to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestCheckBlockClearsUnusedSide(t *testing.T) {
	reps := uuid.New()
	lk := trainingLookups{
		kinds: map[string]database.ListBlockKindsRow{
			"single":   {Code: "single", Name: "Single", MinExercises: 1},
			"superset": {Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
			"complex":  {Code: "complex", Name: "Complex", MinExercises: 2, SharedSets: true, SharedReps: true, SharedLoad: true},
		},
		repUnits: map[uuid.UUID]unit{reps: {name: "reps"}},
	}
	withReps := amount{RepsValue: ptr(5.0), RepsUnit: &reps, PerSide: true}

	complexBlock := trainingBlockRequest{Kind: "complex", Sets: ptr[int32](5), amount: withReps, Exercises: []trainingExerciseRequest{
		{ExerciseID: uuid.New(), Sets: ptr[int32](4), amount: withReps}, {ExerciseID: uuid.New(), amount: withReps},
	}}
	if msg := checkBlock(&complexBlock, lk); msg != "" {
		t.Fatal(msg)
	}
	for _, e := range complexBlock.Exercises {
		if e.RepsValue != nil || e.PerSide || e.Sets != nil {
			t.Errorf("complex exercise kept its own sets or reps: %+v", e)
		}
	}

	superset := trainingBlockRequest{Kind: "superset", Sets: ptr[int32](3), amount: withReps, Exercises: complexBlock.Exercises}
	for i := range superset.Exercises {
		superset.Exercises[i].amount = withReps
	}
	if msg := checkBlock(&superset, lk); msg != "" {
		t.Fatal(msg)
	}
	if superset.RepsValue != nil {
		t.Errorf("superset block kept shared reps: %+v", superset.amount)
	}

	single := trainingBlockRequest{Kind: "single", Sets: ptr[int32](3), Exercises: []trainingExerciseRequest{
		{ExerciseID: uuid.New(), Sets: ptr[int32](5), amount: withReps},
	}}
	if msg := checkBlock(&single, lk); msg != "" {
		t.Fatal(msg)
	}
	if single.Sets != nil {
		t.Errorf("single block kept shared sets: %d", *single.Sets)
	}
}
