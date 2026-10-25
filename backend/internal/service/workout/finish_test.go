package workout

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func TestFinish(t *testing.T) {
	f, _, _ := newFake()
	f.workout = &repos.Workout{ID: uuid.New(), SessionName: "Session 1"}
	f.sets = []repos.ListWorkoutSetsRow{{ID: uuid.New(), ExerciseName: "Squat", SetNumber: 1}}

	got, err := (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, "  felt strong ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != "felt strong" || len(got.Sets) != 1 || got.Sets[0].ExerciseName != "Squat" {
		t.Errorf("finished = %+v", got)
	}

	_, err = (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, strings.Repeat("x", 2001), nil)
	wantError(t, err, service.KindInvalid, "notes can have at most 2000 characters")

	// Done on another day: the time is passed on; none keeps the stored one (or now).
	yesterday := time.Now().AddDate(0, 0, -1)
	if _, err := (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, "", &yesterday); err != nil {
		t.Fatal(err)
	}
	if got := f.finished[len(f.finished)-1].CompletedAt; !got.Valid || !got.Time.Equal(yesterday) {
		t.Errorf("completed_at = %v, want %v", got, yesterday)
	}
	if f.finished[0].CompletedAt.Valid {
		t.Errorf("completed_at = %v without a date, want none", f.finished[0].CompletedAt)
	}
	tomorrow := time.Now().AddDate(0, 0, 1)
	_, err = (&Service{q: f}).Finish(context.Background(), uuid.New(), f.workout.ID, "", &tomorrow)
	wantError(t, err, service.KindInvalid, "the workout date must be within the last year and not in the future")

	f.workout = nil
	_, err = (&Service{q: f}).Finish(context.Background(), uuid.New(), uuid.New(), "", nil)
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
