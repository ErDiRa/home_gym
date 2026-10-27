//go:build component

package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	planningclient "github.com/erdira/homegym/backend/modules/planning/api/http/client"
	trackingclient "github.com/erdira/homegym/backend/modules/tracking/api/http/client"
)

// thisWeek is the week of the tracker for the current (UTC) week.
func thisWeek(training *uuid.UUID) *trackingclient.GetWeekParams {
	monday := thisMonday()
	return &trackingclient.GetWeekParams{
		Date: openapi_types.Date{Time: monday}, From: monday, To: monday.AddDate(0, 0, 7), Training: training,
	}
}

// activeTraining creates an active 2-week training from this Monday planning exercise and returns
// its id.
func activeTraining(t *testing.T, u user, exercise, reps uuid.UUID) uuid.UUID {
	t.Helper()
	in := trainingInput(thisMonday().Format(time.DateOnly), exercise, reps)
	in.Active = true
	return createTraining(t, u, in)
}

func week(t *testing.T, u user, training *uuid.UUID) trackingclient.Week {
	t.Helper()
	resp, err := u.Tracking.GetWeekWithResponse(t.Context(), thisWeek(training))
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("week: %v %s", err, body(resp.Body))
	}
	return *resp.JSON200
}

// The tracker's flow: the week plans a slot, starting it starts a workout, sets are logged, the
// finished workout marks the slot done and is in the history.
func TestWorkoutFlow(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	training := activeTraining(t, u, exercise, reps)

	w := week(t, u, nil)
	if w.Training == nil || w.Training.Id != training || *w.WeekNumber != 1 || len(w.Planned) != 2 || w.Planned[0].Name != "Session 1" {
		t.Fatalf("week = %+v", w)
	}
	slot := w.Planned[0].Id

	started, err := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: slot})
	if err != nil || started.StatusCode() != http.StatusCreated || started.JSON201.SessionName != "Session 1" {
		t.Fatalf("start = %v %s", err, body(started.Body))
	}
	workout := started.JSON201.Id
	// Starting the slot again returns the workout in progress.
	again, _ := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: slot})
	if again.StatusCode() != http.StatusOK || again.JSON200.Id != workout {
		t.Errorf("start again = %d %s", again.StatusCode(), body(again.Body))
	}

	five := 5.0
	set := trackingclient.SetInput{ExerciseId: exercise, BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: &five, RepsUnit: &reps}
	if resp, err := u.Tracking.LogSetWithResponse(ctx, workout, set); err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("log set = %v %s", err, body(resp.Body))
	}
	finished, err := u.Tracking.FinishWorkoutWithResponse(ctx, workout, trackingclient.FinishWorkoutRequest{Notes: " good "})
	if err != nil || finished.StatusCode() != http.StatusOK || finished.JSON200.CompletedAt == nil || finished.JSON200.Notes != "good" {
		t.Fatalf("finish = %v %s", err, body(finished.Body))
	}
	// The workout shows its set with the catalog's names.
	if s := finished.JSON200.Sets; len(s) != 1 || s[0].ExerciseName != "Squat" || s[0].RepsUnitName != "reps" {
		t.Errorf("sets = %+v", s)
	}

	w = week(t, u, &training)
	if len(w.Done) != 1 || w.Done[0].Id != workout || *w.Done[0].WeekSessionId != slot || w.Done[0].SetCount != 1 {
		t.Errorf("done = %+v", w.Done)
	}
	monday := thisMonday()
	history, _ := u.Tracking.ListWorkoutsWithResponse(ctx, &trackingclient.ListWorkoutsParams{From: monday, To: monday.AddDate(0, 0, 7)})
	if len(*history.JSON200) != 1 {
		t.Errorf("history = %s", body(history.Body))
	}
	stats, _ := u.Tracking.ExerciseStatsWithResponse(ctx)
	if len(*stats.JSON200) != 1 || (*stats.JSON200)[0].Name != "Squat" || (*stats.JSON200)[0].TimesPerformed != 1 {
		t.Errorf("stats = %s", body(stats.Body))
	}
}

func TestWorkoutsAreTheUsersOwn(t *testing.T) {
	t.Parallel()
	u, other := newUser(t), newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	activeTraining(t, u, exercise, reps)
	slot := week(t, u, nil).Planned[0].Id

	// Another user can't start a slot of a training of the user's.
	if resp, _ := other.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: slot}); resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("another user's slot = %d, want 400", resp.StatusCode())
	}
	started, _ := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: slot})
	workout := started.JSON201.Id
	if resp, _ := other.Tracking.GetWorkoutWithResponse(ctx, workout); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("another user's workout = %d, want 404", resp.StatusCode())
	}

	// Sets of another user's exercise are refused.
	five := 5.0
	foreign := createExercise(t, other, exerciseInput("Theirs"))
	set := trackingclient.SetInput{ExerciseId: foreign, BlockPosition: 1, ExercisePosition: 1, SetNumber: 1, RepsValue: &five, RepsUnit: &reps}
	if resp, _ := u.Tracking.LogSetWithResponse(ctx, workout, set); resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != "unknown exercise" {
		t.Errorf("another user's exercise = %d %s", resp.StatusCode(), body(resp.Body))
	}
}

func TestWodScore(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Burpee")), repsUnit(t, u)
	activeTraining(t, u, exercise, reps)
	started, _ := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: week(t, u, nil).Planned[0].Id})
	workout := started.JSON201.Id

	rounds, extra, seconds := int32(7), int32(12), int32(600)
	logged, err := u.Tracking.LogResultWithResponse(ctx, workout, trackingclient.ResultInput{
		BlockPosition: 1, ScoreType: trackingclient.Amrap, Rounds: &rounds, Reps: &extra, TimeSeconds: &seconds,
	})
	// The time isn't part of an AMRAP's score and is dropped.
	if err != nil || logged.StatusCode() != http.StatusOK || logged.JSON200.TimeSeconds != nil || *logged.JSON200.Rounds != 7 {
		t.Fatalf("log result = %v %s", err, body(logged.Body))
	}
	got, _ := u.Tracking.GetWorkoutWithResponse(ctx, workout)
	if len(got.JSON200.Results) != 1 {
		t.Errorf("results = %+v", got.JSON200.Results)
	}
	if resp, _ := u.Tracking.DeleteResultWithResponse(ctx, workout, logged.JSON200.Id); resp.StatusCode() != http.StatusNoContent {
		t.Errorf("delete result = %d", resp.StatusCode())
	}
}

func TestMaxes(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise := createExercise(t, u, exerciseInput("Deadlift"))
	units, _ := u.Catalog.ListLoadUnitsWithResponse(ctx)
	var kg, rpe uuid.UUID
	for _, unit := range *units.JSON200 {
		switch unit.Name {
		case "kg":
			kg = unit.Id
		case "RPE":
			rpe = unit.Id
		}
	}
	today := time.Now().UTC().Format(time.DateOnly)

	added, err := u.Tracking.AddExerciseMaxWithResponse(ctx, exercise, trackingclient.ExerciseMaxInput{LoadValue: 180, LoadUnit: kg, AchievedOn: today})
	if err != nil || added.StatusCode() != http.StatusCreated || added.JSON201.LoadUnitName != "kg" {
		t.Fatalf("add = %v %s", err, body(added.Body))
	}
	resp, _ := u.Tracking.AddExerciseMaxWithResponse(ctx, exercise, trackingclient.ExerciseMaxInput{LoadValue: 8, LoadUnit: rpe, AchievedOn: today})
	if resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != "a 1RM needs a weight unit such as kg or lb" {
		t.Errorf("1RM in RPE = %d %s", resp.StatusCode(), body(resp.Body))
	}
	if resp, _ := u.Tracking.AddExerciseMaxWithResponse(ctx, uuid.New(), trackingclient.ExerciseMaxInput{LoadValue: 100, LoadUnit: kg, AchievedOn: today}); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("1RM of an unknown exercise = %d, want 404", resp.StatusCode())
	}

	list, _ := u.Tracking.ListExerciseMaxesWithResponse(ctx)
	if len(*list.JSON200) != 1 || (*list.JSON200)[0].LoadValue != 180 {
		t.Fatalf("maxes = %s", body(list.Body))
	}
	if resp, _ := u.Tracking.DeleteExerciseMaxWithResponse(ctx, added.JSON201.Id); resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode())
	}
	if list, _ := u.Tracking.ListExerciseMaxesWithResponse(ctx); len(*list.JSON200) != 0 {
		t.Errorf("maxes after delete = %s", body(list.Body))
	}
}

// The week offers the active trainings; a cycle's shows its cycle and goals.
func TestWeekWithCycle(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	monday := thisMonday()
	cycle, _ := u.Planning.CreateCycleWithResponse(ctx, planningclient.CycleInput{
		Name: "Base", StartDate: monday.AddDate(0, 0, -7).Format(time.DateOnly), Weeks: 4,
		Goals: []planningclient.CycleGoalInput{{Title: "Squat 150"}},
	})
	in := trainingInput(monday.Format(time.DateOnly), exercise, reps)
	in.Active, in.CycleId = true, &cycle.JSON201.Id
	createTraining(t, u, in)

	w := week(t, u, nil)
	if c := w.Cycle; c == nil || c.Name != "Base" || *c.WeekNumber != 2 || len(c.Goals) != 1 || *w.Training.CycleName != "Base" {
		t.Errorf("week cycle = %+v, training %+v", w.Cycle, w.Training)
	}
}
