package workout

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func ptr[T any](v T) *T { return &v }

// fakeQueries is an in-memory Queries: it returns the canned rows set on it (pgx.ErrNoRows
// where there are none) and records the writes.
type fakeQueries struct {
	training  *repos.GetActiveTrainingRow
	units     []repos.ListTrainingUnitsRow
	unit      *repos.GetUnitRow
	lastUnit  *uuid.UUID
	workout   *repos.Workout // what GetWorkout and FinishWorkout find
	current   *repos.Workout
	done      []repos.ListWorkoutsRow
	sets      []repos.ListWorkoutSetsRow
	repUnits  []repos.ListRepUnitsRow
	loadUnits []repos.ListLoadUnitsRow

	createErr   error
	foreignExer bool  // UpsertWorkoutSet finds no exercise
	deleteRows  int64 // rows DeleteWorkout and DeleteWorkoutSet affect

	created  []repos.CreateWorkoutParams
	finished []repos.FinishWorkoutParams
	upserts  []repos.UpsertWorkoutSetParams
	listed   []repos.ListWorkoutsParams
}

// newFake returns a fake with a "reps" unit (from 1) and an "RPE" load unit (1-10).
func newFake() (*fakeQueries, uuid.UUID, uuid.UUID) {
	reps, rpe := uuid.New(), uuid.New()
	return &fakeQueries{
		repUnits:  []repos.ListRepUnitsRow{{ID: reps, Name: "reps", MinValue: ptr(1.0)}},
		loadUnits: []repos.ListLoadUnitsRow{{ID: rpe, Name: "RPE", MinValue: ptr(1.0), MaxValue: ptr(10.0)}},
	}, reps, rpe
}

func found[T any](v *T) (T, error) {
	if v == nil {
		var zero T
		return zero, pgx.ErrNoRows
	}
	return *v, nil
}

func (f *fakeQueries) GetActiveTraining(context.Context, uuid.UUID) (repos.GetActiveTrainingRow, error) {
	return found(f.training)
}

func (f *fakeQueries) ListTrainingUnits(context.Context, uuid.UUID) ([]repos.ListTrainingUnitsRow, error) {
	return f.units, nil
}

func (f *fakeQueries) GetUnit(context.Context, repos.GetUnitParams) (repos.GetUnitRow, error) {
	return found(f.unit)
}

func (f *fakeQueries) GetLastUnitID(context.Context, repos.GetLastUnitIDParams) (*uuid.UUID, error) {
	if f.lastUnit == nil {
		return nil, pgx.ErrNoRows
	}
	return f.lastUnit, nil
}

func (f *fakeQueries) CreateWorkout(_ context.Context, arg repos.CreateWorkoutParams) (repos.Workout, error) {
	if f.createErr != nil {
		return repos.Workout{}, f.createErr
	}
	f.created = append(f.created, arg)
	return repos.Workout{ID: uuid.New(), UserID: arg.UserID, TrainingID: arg.TrainingID, UnitID: arg.UnitID, UnitName: arg.UnitName}, nil
}

func (f *fakeQueries) GetWorkout(context.Context, repos.GetWorkoutParams) (repos.Workout, error) {
	return found(f.workout)
}

func (f *fakeQueries) GetCurrentWorkout(context.Context, uuid.UUID) (repos.Workout, error) {
	return found(f.current)
}

func (f *fakeQueries) FinishWorkout(_ context.Context, arg repos.FinishWorkoutParams) (repos.Workout, error) {
	w, err := found(f.workout)
	if err != nil {
		return w, err
	}
	f.finished = append(f.finished, arg)
	w.Notes = arg.Notes
	return w, nil
}

func (f *fakeQueries) DeleteWorkout(context.Context, repos.DeleteWorkoutParams) (int64, error) {
	return f.deleteRows, nil
}

func (f *fakeQueries) ListWorkouts(_ context.Context, arg repos.ListWorkoutsParams) ([]repos.ListWorkoutsRow, error) {
	f.listed = append(f.listed, arg)
	if f.done == nil {
		return []repos.ListWorkoutsRow{}, nil
	}
	return f.done, nil
}

func (f *fakeQueries) ListWorkoutSets(context.Context, uuid.UUID) ([]repos.ListWorkoutSetsRow, error) {
	if f.sets == nil {
		return []repos.ListWorkoutSetsRow{}, nil
	}
	return f.sets, nil
}

func (f *fakeQueries) UpsertWorkoutSet(_ context.Context, arg repos.UpsertWorkoutSetParams) (repos.WorkoutSet, error) {
	if f.foreignExer {
		return repos.WorkoutSet{}, pgx.ErrNoRows
	}
	f.upserts = append(f.upserts, arg)
	return repos.WorkoutSet{ID: uuid.New(), WorkoutID: arg.WorkoutID, ExerciseID: arg.ExerciseID, SetNumber: arg.SetNumber}, nil
}

func (f *fakeQueries) DeleteWorkoutSet(context.Context, repos.DeleteWorkoutSetParams) (int64, error) {
	return f.deleteRows, nil
}

func (f *fakeQueries) ListRepUnits(context.Context) ([]repos.ListRepUnitsRow, error) {
	return f.repUnits, nil
}

func (f *fakeQueries) ListLoadUnits(context.Context) ([]repos.ListLoadUnitsRow, error) {
	return f.loadUnits, nil
}

// wantError fails the test unless err is a *service.Error of kind with message msg.
func wantError(t *testing.T, err error, kind service.Kind, msg string) {
	t.Helper()
	var svcErr *service.Error
	if !errors.As(err, &svcErr) {
		t.Fatalf("err = %v, want a service error %q", err, msg)
	}
	if svcErr.Kind != kind || svcErr.Msg != msg {
		t.Fatalf("err = {kind %d, %q}, want {kind %d, %q}", svcErr.Kind, svcErr.Msg, kind, msg)
	}
}
