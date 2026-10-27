//go:build component

package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	planningclient "github.com/erdira/homegym/backend/modules/planning/api/http/client"
)

// repsUnit returns the id of the rep unit "reps".
func repsUnit(t *testing.T, u user) uuid.UUID {
	t.Helper()
	resp, err := u.Catalog.ListRepUnitsWithResponse(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range *resp.JSON200 {
		if unit.Name == "reps" {
			return unit.Id
		}
	}
	t.Fatal("no rep unit reps")
	return uuid.Nil
}

// singleBlock is a single block of exercise, 3 x 5 reps.
func singleBlock(exercise, reps uuid.UUID) planningclient.TrainingBlockInput {
	sets, five := int32(3), 5.0
	return planningclient.TrainingBlockInput{Kind: "single", Exercises: []planningclient.TrainingExerciseInput{{
		ExerciseId: exercise, Sets: &sets, RepsValue: &five, RepsUnit: &reps,
	}}}
}

// trainingInput is a 2-week training from monday with two sessions of one exercise each: both in
// week 1, the first in week 2.
func trainingInput(monday string, exercise, reps uuid.UUID) planningclient.TrainingInput {
	return planningclient.TrainingInput{
		Name: "Strength", StartDate: monday, Weeks: 2,
		Sessions: []planningclient.TrainingSessionInput{
			{Name: "Session 1", Blocks: []planningclient.TrainingBlockInput{singleBlock(exercise, reps)}},
			{Name: "Session 2", Blocks: []planningclient.TrainingBlockInput{singleBlock(exercise, reps)}},
		},
		Schedule: [][]planningclient.SlotInput{{{Session: 0}, {Session: 1}}, {{Session: 0}}},
	}
}

// thisMonday is the Monday of the current week (UTC), as YYYY-MM-DD.
func thisMonday() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
}

func createTraining(t *testing.T, u user, in planningclient.TrainingInput) uuid.UUID {
	t.Helper()
	resp, err := u.Planning.CreateTrainingWithResponse(t.Context(), in)
	if err != nil || resp.StatusCode() != http.StatusCreated {
		t.Fatalf("create training: %v %s", err, body(resp.Body))
	}
	return resp.JSON201.Id
}

func TestTrainingLifecycle(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	monday := thisMonday().Format(time.DateOnly)
	id := createTraining(t, u, trainingInput(monday, exercise, reps))

	// The detail names the catalog's exercise and unit.
	detail, err := u.Planning.GetTrainingWithResponse(ctx, id)
	if err != nil || detail.StatusCode() != http.StatusOK {
		t.Fatalf("get = %v %s", err, body(detail.Body))
	}
	d := detail.JSON200
	ex := d.Sessions[0].Blocks[0].Exercises[0]
	if ex.Name != "Squat" || ex.Type != planningclient.Strength || ex.RepsUnitName != "reps" || len(d.Schedule) != 2 || len(d.Schedule[0]) != 2 {
		t.Fatalf("detail = %+v", d)
	}

	// An update keeps the ids it sends back and adds a session.
	in := trainingInput(monday, exercise, reps)
	in.Sessions[0].Id, in.Sessions[1].Id = &d.Sessions[0].Id, &d.Sessions[1].Id
	in.Schedule[0][0].Id = &d.Schedule[0][0].Id
	in.Sessions = append(in.Sessions, planningclient.TrainingSessionInput{
		Name: "Session 3", Blocks: []planningclient.TrainingBlockInput{singleBlock(exercise, reps)},
	})
	in.Schedule[1] = append(in.Schedule[1], planningclient.SlotInput{Session: 2})
	if resp, err := u.Planning.UpdateTrainingWithResponse(ctx, id, in); err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("update = %v %s", err, body(resp.Body))
	}
	updated, _ := u.Planning.GetTrainingWithResponse(ctx, id)
	if s := updated.JSON200.Sessions; len(s) != 3 || s[0].Id != d.Sessions[0].Id || updated.JSON200.Schedule[0][0].Id != d.Schedule[0][0].Id {
		t.Errorf("after update = %+v", updated.JSON200)
	}

	// The ids of another training's sessions are refused.
	other := createTraining(t, u, trainingInput(monday, exercise, reps))
	otherDetail, _ := u.Planning.GetTrainingWithResponse(ctx, other)
	in = trainingInput(monday, exercise, reps)
	in.Sessions[0].Id = &otherDetail.JSON200.Sessions[0].Id
	if resp, _ := u.Planning.UpdateTrainingWithResponse(ctx, id, in); resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != "unknown session" {
		t.Errorf("update with another training's session = %d %s", resp.StatusCode(), body(resp.Body))
	}

	// A duplicate is the next phase: as many weeks from the Monday after the end, inactive.
	dup, err := u.Planning.DuplicateTrainingWithResponse(ctx, id)
	if err != nil || dup.StatusCode() != http.StatusCreated {
		t.Fatalf("duplicate = %v %s", err, body(dup.Body))
	}
	if want := thisMonday().AddDate(0, 0, 14); !dup.JSON201.StartDate.Time.Equal(want) || dup.JSON201.Active {
		t.Errorf("duplicate = %+v, want it to start %v", dup.JSON201, want)
	}

	if resp, _ := u.Planning.DeleteTrainingWithResponse(ctx, id); resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode())
	}
	if resp, _ := u.Planning.GetTrainingWithResponse(ctx, id); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode())
	}
	list, _ := u.Planning.ListTrainingsWithResponse(ctx)
	if len(*list.JSON200) != 2 {
		t.Errorf("listed %d trainings, want the other and the duplicate", len(*list.JSON200))
	}
}

// A training can only plan the user's own exercises, in the user's own cycles.
func TestTrainingReferencesAreTheUsers(t *testing.T) {
	t.Parallel()
	u, other := newUser(t), newUser(t)
	ctx := t.Context()
	reps := repsUnit(t, u)
	monday := thisMonday().Format(time.DateOnly)

	foreign := createExercise(t, other, exerciseInput("Not yours"))
	resp, _ := u.Planning.CreateTrainingWithResponse(ctx, trainingInput(monday, foreign, reps))
	if resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != "unknown exercise" {
		t.Errorf("another user's exercise = %d %s", resp.StatusCode(), body(resp.Body))
	}

	cycle, _ := other.Planning.CreateCycleWithResponse(ctx, planningclient.CycleInput{Name: "Theirs", StartDate: monday, Weeks: 4, Goals: []planningclient.CycleGoalInput{}})
	in := trainingInput(monday, createExercise(t, u, exerciseInput("Mine")), reps)
	in.CycleId = &cycle.JSON201.Id
	resp, _ = u.Planning.CreateTrainingWithResponse(ctx, in)
	if resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != "unknown cycle" {
		t.Errorf("another user's cycle = %d %s", resp.StatusCode(), body(resp.Body))
	}
	if resp, _ := u.Planning.GetCycleWithResponse(ctx, cycle.JSON201.Id); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("another user's cycle = %d, want 404", resp.StatusCode())
	}
}

func TestLibrarySession(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Pull-up")), repsUnit(t, u)

	created, err := u.Planning.CreateSessionWithResponse(ctx, planningclient.TrainingSessionInput{
		Name: " Pull day ", Blocks: []planningclient.TrainingBlockInput{singleBlock(exercise, reps)},
	})
	if err != nil || created.StatusCode() != http.StatusCreated || created.JSON201.Name != "Pull day" ||
		created.JSON201.Blocks[0].Exercises[0].Name != "Pull-up" {
		t.Fatalf("create = %v %s", err, body(created.Body))
	}
	id := created.JSON201.Id

	updated, _ := u.Planning.UpdateSessionWithResponse(ctx, id, planningclient.TrainingSessionInput{Name: "Pull day", Blocks: []planningclient.TrainingBlockInput{}})
	if updated.StatusCode() != http.StatusOK || len(updated.JSON200.Blocks) != 0 {
		t.Errorf("update = %d %s", updated.StatusCode(), body(updated.Body))
	}
	if list, _ := u.Planning.ListSessionsWithResponse(ctx); len(*list.JSON200) != 1 || (*list.JSON200)[0].BlockCount != 0 {
		t.Errorf("library = %s", body(list.Body))
	}
	if resp, _ := u.Planning.DeleteSessionWithResponse(ctx, id); resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode())
	}
	if resp, _ := u.Planning.GetSessionWithResponse(ctx, id); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode())
	}
}

func TestCycleWithTrainings(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Deadlift")), repsUnit(t, u)
	monday := thisMonday().Format(time.DateOnly)

	cycle, err := u.Planning.CreateCycleWithResponse(ctx, planningclient.CycleInput{
		Name: "Base", StartDate: monday, Weeks: 8, Goals: []planningclient.CycleGoalInput{{Title: " Pull 200 kg "}},
	})
	if err != nil || cycle.StatusCode() != http.StatusCreated || cycle.JSON201.Goals[0].Title != "Pull 200 kg" || cycle.JSON201.Goals[0].Status != planningclient.Open {
		t.Fatalf("create cycle = %v %s", err, body(cycle.Body))
	}
	cycleID := cycle.JSON201.Id
	in := trainingInput(monday, exercise, reps)
	in.CycleId = &cycleID
	training := createTraining(t, u, in)

	got, _ := u.Planning.GetCycleWithResponse(ctx, cycleID)
	if len(got.JSON200.Trainings) != 1 || got.JSON200.Trainings[0].Id != training {
		t.Errorf("cycle trainings = %+v", got.JSON200.Trainings)
	}

	// Deleting the cycle keeps its trainings by default; they just leave it.
	if resp, _ := u.Planning.DeleteCycleWithResponse(ctx, cycleID, &planningclient.DeleteCycleParams{}); resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("delete cycle = %d", resp.StatusCode())
	}
	detail, _ := u.Planning.GetTrainingWithResponse(ctx, training)
	if detail.StatusCode() != http.StatusOK || detail.JSON200.CycleId != nil {
		t.Errorf("training after its cycle was deleted = %d %+v", detail.StatusCode(), detail.JSON200)
	}
}

func TestImportCycle(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	exercise, reps := createExercise(t, u, exerciseInput("Snatch")), repsUnit(t, u)
	start := thisMonday()

	in := planningclient.CycleImportInput{
		Cycle: planningclient.CycleInput{Name: "Master", StartDate: start.Format(time.DateOnly), Weeks: 3, Goals: []planningclient.CycleGoalInput{}},
	}
	for w := range 3 {
		training := trainingInput(start.AddDate(0, 0, 7*w).Format(time.DateOnly), exercise, reps)
		training.Weeks, training.Schedule = 1, training.Schedule[:1]
		// A long note makes the body larger than the 1 MiB other requests may send.
		training.Description = strings.Repeat("x", 500_000)
		in.Trainings = append(in.Trainings, training)
	}
	imported, err := u.Planning.ImportCycleWithResponse(ctx, in)
	if err != nil || imported.StatusCode() != http.StatusCreated || len(imported.JSON201.Trainings) != 3 {
		t.Fatalf("import = %v %d %s", err, imported.StatusCode(), body(imported.Body))
	}
	got, _ := u.Planning.GetCycleWithResponse(ctx, imported.JSON201.Id)
	if len(got.JSON200.Trainings) != 3 {
		t.Errorf("the imported cycle has %d trainings, want 3", len(got.JSON200.Trainings))
	}

	// A broken training is named, and nothing is imported.
	in.Trainings[1].Sessions[0].Blocks[0].Kind = "foo"
	resp, _ := u.Planning.ImportCycleWithResponse(ctx, in)
	if resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != `Week 2 "Strength": Session 1, block A: unknown block kind` {
		t.Errorf("import of a broken training = %d %s", resp.StatusCode(), body(resp.Body))
	}
	if cycles, _ := u.Planning.ListCyclesWithResponse(ctx); len(*cycles.JSON200) != 1 {
		t.Errorf("%d cycles after a failed import, want 1", len(*cycles.JSON200))
	}
}
