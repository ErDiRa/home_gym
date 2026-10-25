package cycle

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

func TestCreate(t *testing.T) {
	f := &fakeQueries{}
	userID := uuid.New()

	got, err := newTestService(f).Create(context.Background(), userID, validCycle())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.created) != 1 || f.created[0].UserID != userID || got.Name != "Mobility" || got.Weeks != 8 {
		t.Errorf("created = %+v for %+v", got, f.created)
	}
	if start, end := got.StartDate.Time.Format(time.DateOnly), got.EndDate.Time.Format(time.DateOnly); start != "2026-03-02" || end != "2026-04-26" {
		t.Errorf("period = %s to %s, want 2026-03-02 to 2026-04-26", start, end)
	}
	// The goals are written in order, with positions from 1 and open as the default status.
	g := f.insertedGoals
	if len(g) != 2 || g[0].Position != 1 || g[0].Title != "Better hips" || g[0].Status != GoalOpen || g[0].CycleID != got.ID ||
		g[1].Position != 2 || g[1].Status != GoalAchieved {
		t.Errorf("goals = %+v", g)
	}
	if f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls end with %q, want commit", f.calls[len(f.calls)-1])
	}
	if got.Goals == nil || got.Trainings == nil {
		t.Errorf("goals %v and trainings %v must encode as []", got.Goals, got.Trainings)
	}
}

func TestCreateInvalidWritesNothing(t *testing.T) {
	f := &fakeQueries{}
	in := validCycle()
	in.Weeks = 0

	_, err := newTestService(f).Create(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "weeks must be 1-52")
	if f.called("begin") {
		t.Errorf("calls = %v, want no transaction", f.calls)
	}
}

func TestUpdateRewritesGoals(t *testing.T) {
	f := &fakeQueries{}
	f.cycle = &repos.GetCycleRow{Name: "Old"}
	f.trainings = []repos.ListCycleTrainingsRow{{ID: uuid.New(), Name: "Hips"}}
	id, userID := uuid.New(), uuid.New()
	in := validCycle()
	in.Goals = in.Goals[1:] // the first goal is dropped

	got, err := newTestService(f).Update(context.Background(), userID, id, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.updated) != 1 || f.updated[0].ID != id || f.updated[0].UserID != userID || got.Name != "Mobility" {
		t.Errorf("updated = %+v", f.updated)
	}
	// The goals are cleared first and written again, as sent.
	if del, ins := f.indexOf("DeleteCycleGoals"), f.indexOf("InsertCycleGoal"); del < 0 || del > ins || f.clearedGoalsOf[0] != id {
		t.Errorf("calls = %v", f.calls)
	}
	if len(f.insertedGoals) != 1 || f.insertedGoals[0].Title != "Better elbow extension" || f.insertedGoals[0].Position != 1 {
		t.Errorf("goals = %+v", f.insertedGoals)
	}
	if len(got.Trainings) != 1 || got.Trainings[0].Name != "Hips" {
		t.Errorf("trainings = %+v", got.Trainings)
	}
}

// Another user's cycle (or a deleted one) updates nothing and is not found.
func TestUpdateForeign(t *testing.T) {
	f := &fakeQueries{}

	_, err := newTestService(f).Update(context.Background(), uuid.New(), uuid.New(), validCycle())
	wantError(t, err, service.KindNotFound, "cycle not found")
	if f.called("InsertCycleGoal") || f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls = %v", f.calls)
	}
}

func TestGet(t *testing.T) {
	f := &fakeQueries{}
	f.cycle = &repos.GetCycleRow{Name: "Mobility"}
	f.goals = []repos.ListCycleGoalsRow{{Title: "Better hips", Position: 1}}
	id := uuid.New()

	got, err := newTestService(f).Get(context.Background(), uuid.New(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id || len(got.Goals) != 1 || got.Trainings == nil {
		t.Errorf("got = %+v", got)
	}

	f.cycle = nil
	_, err = newTestService(f).Get(context.Background(), uuid.New(), id)
	wantError(t, err, service.KindNotFound, "cycle not found")
}
