package training

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
)

// twoUnitInput is a valid training with two units: a single block with two exercises, and a
// superset.
func twoUnitInput(reps uuid.UUID) Input {
	withReps := Amount{RepsValue: ptr(8.0), RepsUnit: &reps}
	return Input{ClientID: uuid.New(), Name: "Strength", Units: []UnitInput{
		{Name: "Day A", Blocks: []BlockInput{{Kind: "single", Exercises: []ExerciseInput{
			{ExerciseID: uuid.New(), Sets: ptr[int32](3), Amount: withReps},
			{ExerciseID: uuid.New(), Sets: ptr[int32](4), Amount: withReps},
		}}}},
		{Name: "Day B", Blocks: []BlockInput{{Kind: "superset", Sets: ptr[int32](3), Exercises: []ExerciseInput{
			{ExerciseID: uuid.New(), Amount: withReps},
			{ExerciseID: uuid.New(), Amount: withReps},
		}}}},
	}}
}

func TestCreate(t *testing.T) {
	f, reps, _ := newFake()
	in := twoUnitInput(reps)

	got, err := newTestService(f).Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != in.ClientID || got.Name != "Strength" || got.Active {
		t.Errorf("created = %+v", got)
	}
	if f.called("DeactivateClientTrainings") {
		t.Error("an inactive training deactivated the client's trainings")
	}
	if f.calls[len(f.calls)-1] != "commit" {
		t.Errorf("calls end with %q, want commit", f.calls[len(f.calls)-1])
	}

	// Positions follow the input order within each parent.
	if len(f.insertedUnits) != 2 || f.insertedUnits[0].Position != 1 || f.insertedUnits[1].Position != 2 || f.insertedUnits[1].Name != "Day B" {
		t.Errorf("units = %+v", f.insertedUnits)
	}
	if len(f.insertedBlocks) != 2 || f.insertedBlocks[0].Position != 1 || f.insertedBlocks[1].Position != 1 {
		t.Errorf("blocks = %+v", f.insertedBlocks)
	}
	if len(f.insertedExercises) != 4 || f.insertedExercises[1].Position != 2 || *f.insertedExercises[1].Sets != 4 {
		t.Errorf("exercises = %+v", f.insertedExercises)
	}
	// The superset shares its sets: they sit on the block, not on its exercises.
	if f.insertedBlocks[1].Sets == nil || f.insertedExercises[2].Sets != nil {
		t.Errorf("superset sets: block %v, exercise %v", f.insertedBlocks[1].Sets, f.insertedExercises[2].Sets)
	}
}

func TestCreateActiveDeactivatesFirst(t *testing.T) {
	f, reps, _ := newFake()
	in := twoUnitInput(reps)
	in.Active = true

	if _, err := newTestService(f).Create(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	deactivate, create := f.indexOf("DeactivateClientTrainings"), f.indexOf("CreateTraining")
	if deactivate < 0 || deactivate > create {
		t.Fatalf("calls = %v, want DeactivateClientTrainings before CreateTraining", f.calls)
	}
	if f.deactivated[0] != in.ClientID {
		t.Errorf("deactivated client %v, want %v", f.deactivated[0], in.ClientID)
	}
}

func TestCreateInvalidWritesNothing(t *testing.T) {
	f, reps, _ := newFake()
	in := twoUnitInput(reps)
	in.Name = ""

	_, err := newTestService(f).Create(context.Background(), in)
	wantError(t, err, service.KindInvalid, "name is required (max 100 characters)")
	if f.called("begin") {
		t.Errorf("calls = %v, want no transaction", f.calls)
	}
}

func TestCreateUnknownClient(t *testing.T) {
	f, reps, _ := newFake()
	f.clientExists = false

	_, err := newTestService(f).Create(context.Background(), twoUnitInput(reps))
	wantError(t, err, service.KindInvalid, "unknown client")
	if f.called("CreateTraining") {
		t.Error("created a training for an unknown client")
	}
}

func TestCreateUnknownExercise(t *testing.T) {
	f, reps, _ := newFake()
	f.insertExerErr = &pgconn.PgError{Code: "23503"}

	_, err := newTestService(f).Create(context.Background(), twoUnitInput(reps))
	wantError(t, err, service.KindInvalid, "unknown exercise")
	if f.calls[len(f.calls)-1] != "rollback" {
		t.Errorf("calls end with %q, want rollback", f.calls[len(f.calls)-1])
	}
}
