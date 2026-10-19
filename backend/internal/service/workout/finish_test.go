package workout

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func TestFinish(t *testing.T) {
	f, _, _ := newFake()
	f.workout = &repos.Workout{ID: uuid.New(), SessionName: "Session 1"}
	f.sets = []repos.ListWorkoutSetsRow{{ID: uuid.New(), ExerciseName: "Squat", SetNumber: 1}}

	got, err := (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, "  felt strong ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != "felt strong" || len(got.Sets) != 1 || got.Sets[0].ExerciseName != "Squat" {
		t.Errorf("finished = %+v", got)
	}

	_, err = (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, strings.Repeat("x", 2001))
	wantError(t, err, service.KindInvalid, "notes can have at most 2000 characters")

	f.workout = nil
	_, err = (&Service{q: f}).Finish(context.Background(), uuid.New(), uuid.New(), "")
	wantError(t, err, service.KindNotFound, "workout not found")
}

func TestDelete(t *testing.T) {
	f, _, _ := newFake()
	f.deleteRows = 1
	if err := (&Service{q: f}).Delete(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatal(err)
	}

	f.deleteRows = 0
	err := (&Service{q: f}).Delete(context.Background(), uuid.New(), uuid.New())
	wantError(t, err, service.KindNotFound, "workout not found")
}
