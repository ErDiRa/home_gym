package training

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func TestCheckBlock(t *testing.T) {
	reps, rpe := uuid.New(), uuid.New()
	lk := lookups{
		kinds: map[string]repos.ListBlockKindsRow{
			"single":   {Code: "single", Name: "Single", MinExercises: 1},
			"pair":     {Code: "pair", Name: "Pair", MinExercises: 2, MaxExercises: ptr[int32](2), SharedSets: true},
			"superset": {Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
			"complex":  {Code: "complex", Name: "Complex", MinExercises: 2, SharedSets: true, SharedReps: true, SharedLoad: true},
		},
		repUnits:  map[uuid.UUID]unit{reps: {name: "reps", min: ptr(1.0)}},
		loadUnits: map[uuid.UUID]unit{rpe: {name: "RPE", min: ptr(1.0), max: ptr(10.0)}},
	}
	withReps := Amount{RepsValue: ptr(8.0), RepsUnit: &reps}
	exercise := func(a Amount) ExerciseInput {
		return ExerciseInput{ExerciseID: uuid.New(), Sets: ptr[int32](3), Amount: a}
	}

	tests := []struct {
		name    string
		block   BlockInput
		wantErr string // substring; empty means valid
	}{
		{"single", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(withReps)}}, ""},
		{"single with two exercises", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(withReps), exercise(withReps)}}, ""},
		{"single without exercises", BlockInput{Kind: "single"}, "at least 1 exercise"},
		{"pair with three exercises", BlockInput{Kind: "pair", Sets: ptr[int32](3), Exercises: []ExerciseInput{exercise(withReps), exercise(withReps), exercise(withReps)}}, "exactly 2"},
		{"superset with one exercise", BlockInput{Kind: "superset", Sets: ptr[int32](3), Exercises: []ExerciseInput{exercise(withReps)}}, "at least 2"},
		{"unknown kind", BlockInput{Kind: "circuit", Sets: ptr[int32](3)}, "unknown block kind"},
		{"superset without sets", BlockInput{Kind: "superset", Exercises: []ExerciseInput{exercise(withReps), exercise(withReps)}}, "sets must be"},
		{"single exercise without sets", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(withReps), {ExerciseID: uuid.New(), Amount: withReps}}}, "exercise 2: sets must be"},
		{"exercise without reps", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(Amount{})}}, "reps are required"},
		{"complex needs block reps", BlockInput{Kind: "complex", Sets: ptr[int32](5), Exercises: []ExerciseInput{exercise(Amount{}), exercise(Amount{})}}, "reps are required"},
		{"complex", BlockInput{Kind: "complex", Sets: ptr[int32](5), Amount: withReps, Exercises: []ExerciseInput{exercise(Amount{}), exercise(Amount{})}}, ""},
		{"range must rise", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(Amount{RepsValue: ptr(10.0), RepsMax: ptr(8.0), RepsUnit: &reps})}}, "higher"},
		{"load out of unit bounds", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(Amount{RepsValue: ptr(8.0), RepsUnit: &reps, LoadValue: ptr(12.0), LoadUnit: &rpe})}}, "between 1 and 10"},
		{"load without unit", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(Amount{RepsValue: ptr(8.0), RepsUnit: &reps, LoadValue: ptr(80.0)})}}, "value and a unit"},
		{"unknown unit", BlockInput{Kind: "single", Exercises: []ExerciseInput{exercise(Amount{RepsValue: ptr(8.0), RepsUnit: ptr(uuid.New())})}}, "unknown reps unit"},
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
	lk := lookups{
		kinds: map[string]repos.ListBlockKindsRow{
			"single":   {Code: "single", Name: "Single", MinExercises: 1},
			"superset": {Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
			"complex":  {Code: "complex", Name: "Complex", MinExercises: 2, SharedSets: true, SharedReps: true, SharedLoad: true},
		},
		repUnits: map[uuid.UUID]unit{reps: {name: "reps"}},
	}
	withReps := Amount{RepsValue: ptr(5.0), RepsUnit: &reps, PerSide: true}

	complexBlock := BlockInput{Kind: "complex", Sets: ptr[int32](5), Amount: withReps, Exercises: []ExerciseInput{
		{ExerciseID: uuid.New(), Sets: ptr[int32](4), Amount: withReps}, {ExerciseID: uuid.New(), Amount: withReps},
	}}
	if msg := checkBlock(&complexBlock, lk); msg != "" {
		t.Fatal(msg)
	}
	for _, e := range complexBlock.Exercises {
		if e.RepsValue != nil || e.PerSide || e.Sets != nil {
			t.Errorf("complex exercise kept its own sets or reps: %+v", e)
		}
	}

	superset := BlockInput{Kind: "superset", Sets: ptr[int32](3), Amount: withReps, Exercises: complexBlock.Exercises}
	for i := range superset.Exercises {
		superset.Exercises[i].Amount = withReps
	}
	if msg := checkBlock(&superset, lk); msg != "" {
		t.Fatal(msg)
	}
	if superset.RepsValue != nil {
		t.Errorf("superset block kept shared reps: %+v", superset.Amount)
	}

	single := BlockInput{Kind: "single", Sets: ptr[int32](3), Exercises: []ExerciseInput{
		{ExerciseID: uuid.New(), Sets: ptr[int32](5), Amount: withReps},
	}}
	if msg := checkBlock(&single, lk); msg != "" {
		t.Fatal(msg)
	}
	if single.Sets != nil {
		t.Errorf("single block kept shared sets: %d", *single.Sets)
	}
}

func TestValidate(t *testing.T) {
	lk := lookups{kinds: map[string]repos.ListBlockKindsRow{"single": {Code: "single", Name: "Single", MinExercises: 1}}}
	sessionID, slotID := uuid.New(), uuid.New()
	valid := func() Input {
		return Input{Name: "  Strength  ", StartDate: "2026-03-02", Weeks: 2, Sessions: []SessionInput{
			{Name: " Session 1 "}, {Name: "Session 2"},
		}, Schedule: [][]SlotInput{{{Session: 0}, {Session: 1}}, {}}}
	}
	week := func(n int) []SlotInput { return make([]SlotInput, n) } // n times session 0

	tests := []struct {
		name    string
		edit    func(*Input)
		wantMsg string // empty means valid
	}{
		{"valid", func(*Input) {}, ""},
		{"blank name", func(in *Input) { in.Name = "   " }, "name is required (max 100 characters)"},
		{"long name", func(in *Input) { in.Name = strings.Repeat("ä", 101) }, "name is required (max 100 characters)"},
		{"session without name", func(in *Input) { in.Sessions[0].Name = "" }, "every session needs a name (max 100 characters)"},
		{"session twice", func(in *Input) {
			in.Sessions = []SessionInput{{ID: &sessionID, Name: "A"}, {ID: &sessionID, Name: "B"}}
		}, "a session appears twice"},
		{"missing start", func(in *Input) { in.StartDate = "" }, "start_date is required (YYYY-MM-DD)"},
		{"bad start", func(in *Input) { in.StartDate = "02.03.2026" }, "start_date is required (YYYY-MM-DD)"},
		{"start on a Sunday", func(in *Input) { in.StartDate = "2026-03-01" }, "start_date must be a Monday"},
		{"no weeks", func(in *Input) { in.Weeks = 0 }, "weeks must be 1-52"},
		{"too many weeks", func(in *Input) { in.Weeks = 53 }, "weeks must be 1-52"},
		{"schedule shorter than the weeks", func(in *Input) { in.Weeks = 3 }, "the schedule needs one list of sessions per week (3)"},
		{"seven sessions a week", func(in *Input) { in.Schedule[1] = week(7) }, ""},
		{"eight sessions a week", func(in *Input) { in.Schedule[1] = week(8) }, "week 2: at most 7 sessions a week"},
		{"unknown session index", func(in *Input) { in.Schedule[0][1].Session = 2 }, "week 1: unknown session"},
		{"negative session index", func(in *Input) { in.Schedule[0][1].Session = -1 }, "week 1: unknown session"},
		{"slot twice", func(in *Input) {
			in.Schedule[0][0].ID, in.Schedule[0][1].ID = &slotID, &slotID
		}, "a slot of the schedule appears twice"},
		{"unplanned session", func(in *Input) { in.Schedule = [][]SlotInput{{}, {}} }, ""},
		{"no sessions", func(in *Input) { in.Sessions, in.Schedule = nil, [][]SlotInput{{}, {}} }, ""},
		{"bad block names session and letter", func(in *Input) {
			in.Sessions[0].Blocks = []BlockInput{{Kind: "single"}}
		}, "Session 1, block A: a Single block needs at least 1 exercise"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.edit(&in)
			err := validate(&in, lk)
			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			wantError(t, err, service.KindInvalid, tt.wantMsg)
		})
	}

	t.Run("normalizes", func(t *testing.T) {
		in := valid()
		if err := validate(&in, lk); err != nil {
			t.Fatal(err)
		}
		if s := in.Sessions[0]; in.Name != "Strength" || s.Name != "Session 1" {
			t.Errorf("names not trimmed: %q, %q", in.Name, s.Name)
		}
	})
}
