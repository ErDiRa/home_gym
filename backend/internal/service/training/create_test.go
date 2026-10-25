package training

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// twoSessionInput is a valid two-week training with two sessions, a single block with two
// exercises and a superset, planned as Session 1, Session 2, Session 1 in week 1 and Session 2
// alone in week 2.
func twoSessionInput(reps uuid.UUID) Input {
	withReps := Amount{RepsValue: ptr(8.0), RepsUnit: &reps}
	return Input{Name: "Strength", StartDate: "2026-03-02", Weeks: 2, Sessions: []SessionInput{
		{Name: "Session 1", Blocks: []BlockInput{{Kind: "single", Exercises: []ExerciseInput{
			{ExerciseID: uuid.New(), Sets: ptr[int32](3), Amount: withReps},
			{ExerciseID: uuid.New(), Sets: ptr[int32](4), Amount: withReps},
		}}}},
		{Name: "Session 2", Blocks: []BlockInput{{Kind: "superset", Sets: ptr[int32](3), Exercises: []ExerciseInput{
			{ExerciseID: uuid.New(), Amount: withReps},
			{ExerciseID: uuid.New(), Amount: withReps},
		}}}},
	}, Schedule: [][]SlotInput{{{Session: 0}, {Session: 1}, {Session: 0}}, {{Session: 1}}}}
}

func TestCreate(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	userID := uuid.New()

	got, err := newTestService(f).Create(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	if f.created[0].UserID != userID || got.Name != "Strength" || got.Active || got.Weeks != 2 {
		t.Errorf("created = %+v for %v", got, f.created[0].UserID)
	}
	if start, end := got.StartDate.Time.Format(time.DateOnly), got.EndDate.Time.Format(time.DateOnly); start != "2026-03-02" || end != "2026-03-15" {
		t.Errorf("period = %s to %s, want 2026-03-02 to 2026-03-15", start, end)
	}
	if f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls end with %q, want commit", f.calls[len(f.calls)-1])
	}

	// Positions follow the input order within each parent; the sessions belong to the training.
	ss := f.insertedSessions
	if len(ss) != 2 || ss[0].Position != 1 || ss[1].Position != 2 || ss[1].Name != "Session 2" || *ss[0].TrainingID != got.ID || ss[0].UserID != userID {
		t.Errorf("sessions = %+v", ss)
	}
	if len(f.insertedBlocks) != 2 || f.insertedBlocks[0].Position != 1 || f.insertedBlocks[1].Position != 1 || f.insertedBlocks[1].SessionID != f.sessionIDs[1] {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
	if len(f.insertedExercises) != 4 || f.insertedExercises[1].Position != 2 || *f.insertedExercises[1].Sets != 4 || f.insertedExercises[0].UserID != userID {
		t.Errorf("exercises = %+v", f.insertedExercises)
	}
	// The superset shares its sets: they sit on the block, not on its exercises.
	if f.insertedBlocks[1].Sets == nil || f.insertedExercises[2].Sets != nil {
		t.Errorf("superset sets: block %v, exercise %v", f.insertedBlocks[1].Sets, f.insertedExercises[2].Sets)
	}

	// The schedule points at the saved sessions, by week and position.
	s1, s2 := f.sessionIDs[0], f.sessionIDs[1]
	want := []repos.InsertWeekSessionParams{
		{TrainingID: got.ID, Week: 1, Position: 1, SessionID: s1},
		{TrainingID: got.ID, Week: 1, Position: 2, SessionID: s2},
		{TrainingID: got.ID, Week: 1, Position: 3, SessionID: s1},
		{TrainingID: got.ID, Week: 2, Position: 1, SessionID: s2},
	}
	if !slices.Equal(f.insertedSlots, want) {
		t.Errorf("slots = %+v, want %+v", f.insertedSlots, want)
	}
}

// Several trainings can be active: saving one active leaves the others alone.
func TestCreateActive(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	in.Active = true

	got, err := newTestService(f).Create(context.Background(), uuid.New(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Active || !f.created[0].Active {
		t.Errorf("created = %+v", got)
	}
}

func TestCreateInvalidWritesNothing(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	in.Name = ""

	_, err := newTestService(f).Create(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "name is required (max 100 characters)")
	if f.called("begin") {
		t.Errorf("calls = %v, want no transaction", f.calls)
	}
}

func TestCreateForeignExercise(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	f.foreignExer[in.Sessions[1].Blocks[0].Exercises[0].ExerciseID] = true // another user's

	_, err := newTestService(f).Create(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown exercise")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}

func TestCreateUnknownExercise(t *testing.T) {
	f, reps, _ := newFake()
	f.insertExerErr = &pgconn.PgError{Code: "23503"}

	_, err := newTestService(f).Create(context.Background(), uuid.New(), twoSessionInput(reps))
	wantError(t, err, service.KindInvalid, "unknown exercise")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}

func TestCreateInCycle(t *testing.T) {
	f, reps, _ := newFake()
	cycleID, userID := uuid.New(), uuid.New()
	f.ownedCycles[cycleID] = true
	in := twoSessionInput(reps)
	in.CycleID = &cycleID

	if _, err := newTestService(f).Create(context.Background(), userID, in); err != nil {
		t.Fatal(err)
	}
	if f.created[0].CycleID == nil || *f.created[0].CycleID != cycleID {
		t.Errorf("created in cycle %v, want %v", f.created[0].CycleID, cycleID)
	}
}

// Another user's (or a deleted) cycle is unknown, and nothing is written.
func TestCreateInForeignCycle(t *testing.T) {
	f, reps, _ := newFake()
	in := twoSessionInput(reps)
	in.CycleID = ptr(uuid.New())

	_, err := newTestService(f).Create(context.Background(), uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown cycle")
	if f.called("CreateTraining") || f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls = %v", f.calls)
	}
}
