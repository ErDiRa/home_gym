package app

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

var (
	// monday is a Monday (UTC midnight, as the week's date parses), from its week's bounds.
	monday   = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	from, to = monday, monday.AddDate(0, 0, 7)
)

func TestWeekNumber(t *testing.T) {
	start := monday.AddDate(0, 0, -14) // week 3 of 4
	if n := weekNumber(monday, start, 4); n == nil || *n != 3 {
		t.Errorf("week = %v, want 3", n)
	}
	for _, m := range []time.Time{start.AddDate(0, 0, -7), start.AddDate(0, 0, 28)} {
		if n := weekNumber(m, start, 4); n != nil {
			t.Errorf("week of %v = %d, want none outside the training", m, *n)
		}
	}
}

func TestWeekPicksTraining(t *testing.T) {
	f := newFixture()
	a := planning.ActiveTraining{ID: uuid.New(), Name: "A", StartDate: monday, Weeks: 4}
	b := planning.ActiveTraining{ID: uuid.New(), Name: "B", StartDate: monday.AddDate(0, 0, -7), Weeks: 4}
	f.planning.trainings = []planning.ActiveTraining{a, b}
	slot := planning.PlannedSlot{ID: uuid.New(), SessionName: "Session 1"}
	f.planning.plans[b.ID] = map[int32][]planning.PlannedSlot{2: {slot}}
	f.repo.done = []WorkoutSummary{{ID: uuid.New()}}

	week, err := f.s.Week(context.Background(), uuid.New(), b.ID, monday, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if week.Training.ID != b.ID || *week.WeekNumber != 2 || !slices.Equal(week.Planned, []planning.PlannedSlot{slot}) || len(week.Done) != 1 {
		t.Errorf("week = %+v", week)
	}

	// Without a choice (or an inactive one) the first is shown.
	week, _ = f.s.Week(context.Background(), uuid.New(), uuid.New(), monday, from, to)
	if week.Training.ID != a.ID || *week.WeekNumber != 1 || len(week.Planned) != 0 {
		t.Errorf("default week = %+v", week)
	}
}

func TestWeekWithoutTraining(t *testing.T) {
	f := newFixture()
	week, err := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, from, to)
	if err != nil || week.Training != nil || week.WeekNumber != nil || week.Planned == nil {
		t.Errorf("week = %+v, %v", week, err)
	}
}

func TestWeekRejects(t *testing.T) {
	f := newFixture()
	_, err := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday.AddDate(0, 0, 1), from, to)
	wantError(t, err, errs.KindInvalid, "date is required (YYYY-MM-DD), a Monday")
	_, err = f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, time.Time{}, to)
	wantError(t, err, errs.KindInvalid, "from and to are required (RFC 3339), from before to")
	wantError(t, CheckPeriod(from, from.AddDate(2, 0, 0)), errs.KindInvalid, "the period can be at most a year")
}

// A cycle planned week by week has many active one-week trainings: the week offers the ones
// running in it, plus the trainings without a cycle; if none would be left, all.
func TestWeekOffersCycleTrainingsOfTheWeek(t *testing.T) {
	f := newFixture()
	cycle := uuid.New()
	thisWeek := planning.ActiveTraining{ID: uuid.New(), Name: "Week 2", StartDate: monday, Weeks: 1, CycleID: &cycle}
	lastWeek := planning.ActiveTraining{ID: uuid.New(), Name: "Week 1", StartDate: monday.AddDate(0, 0, -7), Weeks: 1, CycleID: &cycle}
	own := planning.ActiveTraining{ID: uuid.New(), Name: "Mobility", StartDate: monday.AddDate(0, 0, 70), Weeks: 4}
	f.planning.trainings = []planning.ActiveTraining{lastWeek, thisWeek, own}

	week, _ := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, from, to)
	if ids := []uuid.UUID{week.Trainings[0].ID, week.Trainings[1].ID}; len(week.Trainings) != 2 || ids[0] != thisWeek.ID || ids[1] != own.ID {
		t.Errorf("offered %+v", week.Trainings)
	}

	f.planning.trainings = []planning.ActiveTraining{lastWeek}
	if week, _ := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, from, to); len(week.Trainings) != 1 || week.WeekNumber != nil {
		t.Errorf("when none runs this week: %+v", week)
	}
}

func TestWeekCycle(t *testing.T) {
	f := newFixture()
	cycleID := uuid.New()
	f.planning.trainings = []planning.ActiveTraining{{ID: uuid.New(), StartDate: monday, Weeks: 1, CycleID: &cycleID}}
	f.planning.cycles[cycleID] = planning.Cycle{ID: cycleID, Name: "Base", StartDate: monday.AddDate(0, 0, -21), Weeks: 8,
		Goals: []planning.Goal{{Title: "Squat 150"}}}

	week, _ := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, from, to)
	if c := week.Cycle; c == nil || c.Name != "Base" || *c.WeekNumber != 4 || c.Goals[0].Title != "Squat 150" {
		t.Errorf("cycle = %+v", c)
	}

	// A cycle deleted meanwhile is none.
	delete(f.planning.cycles, cycleID)
	if week, err := f.s.Week(context.Background(), uuid.New(), uuid.Nil, monday, from, to); err != nil || week.Cycle != nil {
		t.Errorf("cycle = %+v, %v", week.Cycle, err)
	}
}

func TestStartWorkout(t *testing.T) {
	f := newFixture()
	f.planning.slot = planning.Slot{ID: uuid.New(), TrainingID: uuid.New(), SessionID: uuid.New(), SessionName: "Session 1"}

	_, created, err := f.s.StartWorkout(context.Background(), uuid.New(), f.planning.slot.ID)
	if err != nil || !created || f.repo.created[0] != f.planning.slot {
		t.Errorf("start = %v, %v, saved %+v", created, err, f.repo.created)
	}

	// A slot already in progress returns its workout.
	f.repo.inProgress = true
	id, created, err := f.s.StartWorkout(context.Background(), uuid.New(), f.planning.slot.ID)
	if err != nil || created || id != f.repo.workout {
		t.Errorf("start of a slot in progress = %v, %v, %v", id, created, err)
	}

	_, _, err = f.s.StartWorkout(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, errs.KindInvalid, "unknown planned session")
}

func TestFinishWorkout(t *testing.T) {
	f := newFixture()
	if err := f.s.FinishWorkout(context.Background(), uuid.New(), f.repo.workout, " good ", nil); err != nil || f.repo.finished[0] != "good" {
		t.Errorf("finish = %v, notes %q", err, f.repo.finished)
	}
	future := f.s.now().Add(time.Hour)
	err := f.s.FinishWorkout(context.Background(), uuid.New(), f.repo.workout, "", &future)
	wantError(t, err, errs.KindInvalid, "the workout date must be within the last year and not in the future")
	err = f.s.FinishWorkout(context.Background(), uuid.New(), uuid.New(), "", nil)
	wantError(t, err, errs.KindNotFound, "workout not found")
}

func TestLogSet(t *testing.T) {
	f := newFixture()
	valid := func() SetInput {
		return SetInput{ExerciseID: f.exercise, BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: ptr(5.0), RepsUnit: &f.catalog.reps}
	}
	if _, err := f.s.LogSet(context.Background(), uuid.New(), f.repo.workout, valid()); err != nil {
		t.Fatal(err)
	}
	// Only the load, e.g. calories on a machine.
	loadOnly := valid()
	loadOnly.RepsValue, loadOnly.LoadValue, loadOnly.LoadUnit = nil, ptr(80.0), &f.catalog.kg
	if _, err := f.s.LogSet(context.Background(), uuid.New(), f.repo.workout, loadOnly); err != nil {
		t.Fatal(err)
	}
	// On bodyweight with weight added, e.g. a weighted pull-up.
	weighted := valid()
	weighted.Bodyweight, weighted.LoadValue, weighted.LoadUnit = true, ptr(10.0), &f.catalog.kg
	if _, err := f.s.LogSet(context.Background(), uuid.New(), f.repo.workout, weighted); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		edit     func(*SetInput)
		workout  uuid.UUID
		wantKind errs.Kind
		wantMsg  string
	}{
		{"no exercise", func(in *SetInput) { in.ExerciseID = uuid.Nil }, f.repo.workout, errs.KindInvalid, "exercise_id is required"},
		{"no block", func(in *SetInput) { in.BlockPosition = 0 }, f.repo.workout, errs.KindInvalid, "block_position and exercise_position must be 1 or more"},
		{"set 0", func(in *SetInput) { in.SetNumber = 0 }, f.repo.workout, errs.KindInvalid, "set_number must be 1-100"},
		{"no reps unit", func(in *SetInput) { in.RepsUnit = nil }, f.repo.workout, errs.KindInvalid, "reps_unit is required"},
		{"nothing entered", func(in *SetInput) { in.RepsValue = nil }, f.repo.workout, errs.KindInvalid, "enter the reps or the load"},
		{"load without unit", func(in *SetInput) { in.LoadValue = ptr(10.0) }, f.repo.workout, errs.KindInvalid, "load needs a value and a unit"},
		{"unknown reps unit", func(in *SetInput) { in.RepsUnit = ptr(uuid.New()) }, f.repo.workout, errs.KindInvalid, "unknown reps unit"},
		{"reps below unit", func(in *SetInput) { in.RepsValue = ptr(0.0) }, f.repo.workout, errs.KindInvalid, "reps in reps must be between 1 and 99999"},
		{"reps above unit", func(in *SetInput) { in.RepsValue, in.RepsUnit = ptr(11.0), &f.catalog.rpe }, f.repo.workout, errs.KindInvalid, "reps in rpe must be between 1 and 10"},
		{"bodyweight plus % 1RM", func(in *SetInput) {
			in.Bodyweight, in.LoadValue, in.LoadUnit = true, ptr(10.0), &f.catalog.percentOfMax
		}, f.repo.workout, errs.KindInvalid, "weight added to bodyweight needs a weight unit such as kg or lb"},
		{"unknown load unit", func(in *SetInput) { in.LoadValue, in.LoadUnit = ptr(11.0), ptr(uuid.New()) }, f.repo.workout, errs.KindInvalid, "unknown load unit"},
		{"another user's workout", func(*SetInput) {}, uuid.New(), errs.KindNotFound, "workout not found"},
		{"another user's exercise", func(in *SetInput) { in.ExerciseID = uuid.New() }, f.repo.workout, errs.KindInvalid, "unknown exercise"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logged := len(f.repo.sets)
			in := valid()
			tt.edit(&in)
			_, err := f.s.LogSet(context.Background(), uuid.New(), tt.workout, in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
			if len(f.repo.sets) != logged {
				t.Error("logged the set anyway")
			}
		})
	}

	// A deleted exercise can't be logged anymore.
	f.catalog.exercises[f.exercise] = catalog.Exercise{ID: f.exercise, Deleted: true}
	_, err := f.s.LogSet(context.Background(), uuid.New(), f.repo.workout, valid())
	wantError(t, err, errs.KindInvalid, "unknown exercise")

	wantError(t, f.s.DeleteSet(context.Background(), uuid.New(), f.repo.workout, uuid.New()), errs.KindNotFound, "set not found")
}

func TestLogResult(t *testing.T) {
	tests := []struct {
		name string
		in   ResultInput
		want ResultInput // with BlockPosition 2
	}{
		{"for time", ResultInput{ScoreType: scoreForTime, TimeSeconds: ptr[int32](522), Rounds: ptr[int32](3), Rx: true},
			ResultInput{ScoreType: scoreForTime, TimeSeconds: ptr[int32](522), Rx: true}},
		{"for time capped", ResultInput{ScoreType: scoreForTime, Reps: ptr[int32](143)},
			ResultInput{ScoreType: scoreForTime, Reps: ptr[int32](143)}},
		{"amrap", ResultInput{ScoreType: scoreAMRAP, Rounds: ptr[int32](5), Reps: ptr[int32](12), TimeSeconds: ptr[int32](720), Notes: "  scaled pull-ups "},
			ResultInput{ScoreType: scoreAMRAP, Rounds: ptr[int32](5), Reps: ptr[int32](12), Notes: "scaled pull-ups"}},
		{"max reps", ResultInput{ScoreType: scoreMaxReps, Reps: ptr[int32](150), Calories: ptr[int32](3)},
			ResultInput{ScoreType: scoreMaxReps, Reps: ptr[int32](150)}},
		{"max calories", ResultInput{ScoreType: scoreMaxCalories, Calories: ptr[int32](42), Reps: ptr[int32](1)},
			ResultInput{ScoreType: scoreMaxCalories, Calories: ptr[int32](42)}},
		{"emom", ResultInput{ScoreType: scoreEMOM, Rounds: ptr[int32](10), Reps: ptr[int32](4)},
			ResultInput{ScoreType: scoreEMOM, Rounds: ptr[int32](10)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			tt.in.BlockPosition, tt.want.BlockPosition = 2, 2
			if _, err := f.s.LogResult(context.Background(), uuid.New(), f.repo.workout, tt.in); err != nil {
				t.Fatal(err)
			}
			if got := f.repo.results[0]; !equalResult(got, tt.want) {
				t.Errorf("logged %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLogResultRejects(t *testing.T) {
	tests := []struct {
		name, msg string
		in        ResultInput
	}{
		{"no block", "block_position must be 1 or more", ResultInput{ScoreType: scoreMaxReps, Reps: ptr[int32](5)}},
		{"unknown format", "unknown score type", ResultInput{BlockPosition: 1, ScoreType: "tabata", Reps: ptr[int32](5)}},
		{"for time without anything", "enter the time, or the reps if the time cap was hit", ResultInput{BlockPosition: 1, ScoreType: scoreForTime}},
		{"for time with time and reps", "enter the time, or the reps if the time cap was hit, not both", ResultInput{BlockPosition: 1, ScoreType: scoreForTime, TimeSeconds: ptr[int32](60), Reps: ptr[int32](5)}},
		{"for time of zero seconds", "the time must be between 1 and 86400", ResultInput{BlockPosition: 1, ScoreType: scoreForTime, TimeSeconds: ptr[int32](0)}},
		{"amrap without rounds", "enter the rounds", ResultInput{BlockPosition: 1, ScoreType: scoreAMRAP, Reps: ptr[int32](12)}},
		{"negative reps", "the reps must be between 0 and 100000", ResultInput{BlockPosition: 1, ScoreType: scoreMaxReps, Reps: ptr[int32](-1)}},
		{"max calories without calories", "enter the calories", ResultInput{BlockPosition: 1, ScoreType: scoreMaxCalories, Reps: ptr[int32](12)}},
		{"emom without rounds", "enter the rounds completed", ResultInput{BlockPosition: 1, ScoreType: scoreEMOM}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			_, err := f.s.LogResult(context.Background(), uuid.New(), f.repo.workout, tt.in)
			wantError(t, err, errs.KindInvalid, tt.msg)
			if len(f.repo.results) != 0 {
				t.Error("logged an invalid result")
			}
		})
	}

	f := newFixture()
	_, err := f.s.LogResult(context.Background(), uuid.New(), uuid.New(), ResultInput{BlockPosition: 1, ScoreType: scoreMaxReps, Reps: ptr[int32](5)})
	wantError(t, err, errs.KindNotFound, "workout not found")
	wantError(t, f.s.DeleteResult(context.Background(), uuid.New(), f.repo.workout, uuid.New()), errs.KindNotFound, "result not found")
}

// equalResult compares results by value, following the pointers.
func equalResult(a, b ResultInput) bool {
	eq := func(x, y *int32) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return a.BlockPosition == b.BlockPosition && a.ScoreType == b.ScoreType &&
		eq(a.TimeSeconds, b.TimeSeconds) && eq(a.Rounds, b.Rounds) && eq(a.Reps, b.Reps) && eq(a.Calories, b.Calories) &&
		a.Rx == b.Rx && a.Notes == b.Notes
}

func TestAddMax(t *testing.T) {
	f := newFixture()
	in := MaxInput{LoadValue: 140, LoadUnit: f.catalog.kg, AchievedOn: "2026-10-13", Notes: " belt "}
	if _, err := f.s.AddMax(context.Background(), uuid.New(), f.exercise, in); err != nil || f.repo.maxes[0].Notes != "belt" {
		t.Fatalf("add = %v, saved %+v", err, f.repo.maxes)
	}

	// From a logged single: the set must be the user's, of that exercise.
	f.repo.ownedSet = uuid.New()
	in.WorkoutSetID = &f.repo.ownedSet
	if _, err := f.s.AddMax(context.Background(), uuid.New(), f.exercise, in); err != nil {
		t.Fatal(err)
	}
	in.WorkoutSetID = ptr(uuid.New())
	_, err := f.s.AddMax(context.Background(), uuid.New(), f.exercise, in)
	wantError(t, err, errs.KindInvalid, "unknown set")

	tests := []struct {
		name string
		in   MaxInput
		msg  string
	}{
		{"zero load", MaxInput{LoadValue: 0, LoadUnit: f.catalog.kg, AchievedOn: "2026-10-13"}, "load must be between 0 and 99999"},
		{"too heavy", MaxInput{LoadValue: 100000, LoadUnit: f.catalog.kg, AchievedOn: "2026-10-13"}, "load must be between 0 and 99999"},
		{"no date", MaxInput{LoadValue: 100, LoadUnit: f.catalog.kg}, "achieved_on must be a date (YYYY-MM-DD)"},
		{"future", MaxInput{LoadValue: 100, LoadUnit: f.catalog.kg, AchievedOn: "2026-10-17"}, "achieved_on must not be in the future"},
		{"percent", MaxInput{LoadValue: 100, LoadUnit: f.catalog.percentOfMax, AchievedOn: "2026-10-13"}, "a 1RM needs a weight unit such as kg or lb"},
		{"unknown unit", MaxInput{LoadValue: 100, LoadUnit: uuid.New(), AchievedOn: "2026-10-13"}, "unknown load unit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.AddMax(context.Background(), uuid.New(), f.exercise, tt.in)
			wantError(t, err, errs.KindInvalid, tt.msg)
		})
	}

	_, err = f.s.AddMax(context.Background(), uuid.New(), uuid.New(), MaxInput{LoadValue: 100, LoadUnit: f.catalog.kg, AchievedOn: "2026-10-13"})
	wantError(t, err, errs.KindNotFound, "exercise not found")
	wantError(t, f.s.DeleteMax(context.Background(), uuid.New(), uuid.New()), errs.KindNotFound, "1RM not found")
}
