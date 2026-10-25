package training

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle"
)

// weeklyCycle is a 10-week cycle import with one one-week training per week, each starting on
// its week's Monday.
func weeklyCycle(reps uuid.UUID) CycleImportInput {
	in := CycleImportInput{Cycle: cycle.Input{Name: "Master", StartDate: "2026-10-26", Weeks: 10,
		Goals: []cycle.GoalInput{{Title: "Peak in week 10"}}}}
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	for w := range 10 {
		t := twoSessionInput(reps)
		t.Name = fmt.Sprintf("Week %d of 10", w+1)
		t.Active = true
		t.StartDate = start.AddDate(0, 0, 7*w).Format(time.DateOnly)
		t.Weeks, t.Schedule = 1, t.Schedule[:1]
		in.Trainings = append(in.Trainings, t)
	}
	return in
}

func TestImportCycle(t *testing.T) {
	f, reps, _ := newFake()
	userID := uuid.New()

	got, err := newTestService(f).ImportCycle(context.Background(), userID, weeklyCycle(reps))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.createdCycles) != 1 || f.createdCycles[0].UserID != userID || got.Name != "Master" || len(f.insertedGoals) != 1 {
		t.Errorf("cycle = %+v, goals %+v", f.createdCycles, f.insertedGoals)
	}
	// Every training goes into the new cycle, with its own start, in one transaction.
	if len(f.created) != 10 {
		t.Fatalf("created %d trainings, want 10", len(f.created))
	}
	for i, c := range f.created {
		want := time.Date(2026, 10, 26+7*i, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
		if c.CycleID == nil || *c.CycleID != got.ID || c.UserID != userID || !c.Active || c.StartDate.Time.Format(time.DateOnly) != want {
			t.Errorf("training %d = %+v, want in cycle %v from %s", i+1, c, got.ID, want)
		}
	}
	if f.calls[0] == "begin" || f.indexOf("begin") > f.indexOf("CreateCycle") || f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls = %v, want lookups, then one transaction", f.calls)
	}
	if len(got.Trainings) != 10 || got.Trainings[6].Name != "Week 7 of 10" {
		t.Errorf("imported trainings = %+v", got.Trainings)
	}
	if n := len(f.insertedSessions); n != 20 {
		t.Errorf("inserted %d sessions, want 20", n)
	}
}

// A broken training is reported with its week and name, and nothing is written.
func TestImportCycleInvalidTraining(t *testing.T) {
	f, reps, _ := newFake()
	in := weeklyCycle(reps)
	in.Trainings[6].Sessions[1].Blocks[0].Kind = "foo"

	_, err := newTestService(f).ImportCycle(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, `Week 7 "Week 7 of 10": Session 2, block A: unknown block kind`)
	if f.called("begin") || f.called("CreateCycle") {
		t.Errorf("calls = %v, want nothing written", f.calls)
	}
}

// A training failing while it is written (another user's exercise) rolls the whole import back.
func TestImportCycleRollsBack(t *testing.T) {
	f, reps, _ := newFake()
	in := weeklyCycle(reps)
	f.foreignExer[in.Trainings[6].Sessions[0].Blocks[0].Exercises[0].ExerciseID] = true

	_, err := newTestService(f).ImportCycle(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, `Week 7 "Week 7 of 10": unknown exercise`)
	if !f.called("CreateCycle") || f.calls[len(f.calls)-1] != "rollback" || f.called("commit") {
		t.Errorf("calls = %v, want a rolled back transaction", f.calls)
	}
}

func TestImportCycleLimits(t *testing.T) {
	f, reps, _ := newFake()
	in := weeklyCycle(reps)
	for len(in.Trainings) <= maxImportTrainings {
		in.Trainings = append(in.Trainings, in.Trainings[0])
	}
	_, err := newTestService(f).ImportCycle(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "a cycle import holds at most 60 trainings")

	// The cycle itself is validated first.
	in = weeklyCycle(reps)
	in.Cycle.StartDate = "2026-10-27"
	_, err = newTestService(f).ImportCycle(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "start_date must be a Monday")
	if f.called("begin") {
		t.Errorf("calls = %v", f.calls)
	}
}
