package training

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

func ptr[T any](v T) *T { return &v }

// fakeQueries is an in-memory Queries: it returns the canned rows and errors set on it and
// records the calls and the parameters of the writes.
type fakeQueries struct {
	calls []string

	// Canned results.
	training      repos.GetTrainingRow
	activeID      *uuid.UUID // nil: no active training
	getErr        error
	units         []repos.ListTrainingUnitsRow
	blocks        []repos.ListTrainingBlocksRow
	exercises     []repos.ListTrainingExercisesRow
	kinds         []repos.ListBlockKindsRow
	repUnits      []repos.ListRepUnitsRow
	loadUnits     []repos.ListLoadUnitsRow
	knownUnits    map[uuid.UUID]bool // unit ids UpdateTrainingUnit finds
	foreignExer   map[uuid.UUID]bool // exercise ids InsertTrainingExercise doesn't find
	updateErr     error
	deleteRows    int64
	insertExerErr error

	// Recorded writes.
	gotTraining       []repos.GetTrainingParams
	created           []repos.CreateTrainingParams
	updated           []repos.UpdateTrainingParams
	deactivated       []uuid.UUID
	keptUnits         []uuid.UUID
	insertedUnits     []repos.InsertTrainingUnitParams
	updatedUnits      []repos.UpdateTrainingUnitParams
	insertedBlocks    []repos.InsertTrainingBlockParams
	insertedExercises []repos.InsertTrainingExerciseParams
}

// newFake returns a fake whose lookups hold a single and a superset kind, a "reps" unit and
// a "kg" unit.
func newFake() (*fakeQueries, uuid.UUID, uuid.UUID) {
	reps, kg := uuid.New(), uuid.New()
	return &fakeQueries{
		kinds: []repos.ListBlockKindsRow{
			{Code: "single", Name: "Single", MinExercises: 1},
			{Code: "superset", Name: "Superset", MinExercises: 2, SharedSets: true},
		},
		repUnits:    []repos.ListRepUnitsRow{{ID: reps, Name: "reps"}},
		loadUnits:   []repos.ListLoadUnitsRow{{ID: kg, Name: "kg"}},
		knownUnits:  map[uuid.UUID]bool{},
		foreignExer: map[uuid.UUID]bool{},
	}, reps, kg
}

// newTestService runs transactions on the fake itself, recording begin/commit/rollback.
func newTestService(f *fakeQueries) *Service {
	return &Service{q: f, inTx: func(_ context.Context, fn func(Queries) error) error {
		f.calls = append(f.calls, "begin")
		err := fn(f)
		if err != nil {
			f.calls = append(f.calls, "rollback")
		} else {
			f.calls = append(f.calls, "commit")
		}
		return err
	}}
}

func (f *fakeQueries) called(name string) bool {
	for _, c := range f.calls {
		if c == name {
			return true
		}
	}
	return false
}

// indexOf returns the position of the first call to name, or -1.
func (f *fakeQueries) indexOf(name string) int {
	for i, c := range f.calls {
		if c == name {
			return i
		}
	}
	return -1
}

func (f *fakeQueries) ListTrainings(context.Context, uuid.UUID) ([]repos.ListTrainingsRow, error) {
	f.calls = append(f.calls, "ListTrainings")
	return []repos.ListTrainingsRow{}, nil
}

func (f *fakeQueries) GetTraining(_ context.Context, arg repos.GetTrainingParams) (repos.GetTrainingRow, error) {
	f.calls = append(f.calls, "GetTraining")
	f.gotTraining = append(f.gotTraining, arg)
	return f.training, f.getErr
}

func (f *fakeQueries) GetActiveTrainingID(context.Context, uuid.UUID) (uuid.UUID, error) {
	f.calls = append(f.calls, "GetActiveTrainingID")
	if f.activeID == nil {
		return uuid.Nil, pgx.ErrNoRows
	}
	return *f.activeID, nil
}

func (f *fakeQueries) ListTrainingUnits(context.Context, uuid.UUID) ([]repos.ListTrainingUnitsRow, error) {
	f.calls = append(f.calls, "ListTrainingUnits")
	return f.units, nil
}

func (f *fakeQueries) ListTrainingBlocks(context.Context, uuid.UUID) ([]repos.ListTrainingBlocksRow, error) {
	f.calls = append(f.calls, "ListTrainingBlocks")
	return f.blocks, nil
}

func (f *fakeQueries) ListTrainingExercises(context.Context, uuid.UUID) ([]repos.ListTrainingExercisesRow, error) {
	f.calls = append(f.calls, "ListTrainingExercises")
	return f.exercises, nil
}

func (f *fakeQueries) CreateTraining(_ context.Context, arg repos.CreateTrainingParams) (repos.CreateTrainingRow, error) {
	f.calls = append(f.calls, "CreateTraining")
	f.created = append(f.created, arg)
	return repos.CreateTrainingRow{ID: uuid.New(), Name: arg.Name, Description: arg.Description, Active: arg.Active,
		StartDate: arg.StartDate, EndDate: arg.EndDate, SessionsPerWeek: arg.SessionsPerWeek}, nil
}

func (f *fakeQueries) UpdateTraining(_ context.Context, arg repos.UpdateTrainingParams) (repos.UpdateTrainingRow, error) {
	f.calls = append(f.calls, "UpdateTraining")
	if f.updateErr != nil {
		return repos.UpdateTrainingRow{}, f.updateErr
	}
	f.updated = append(f.updated, arg)
	return repos.UpdateTrainingRow{ID: arg.ID, Name: arg.Name, Description: arg.Description, Active: arg.Active,
		StartDate: arg.StartDate, EndDate: arg.EndDate, SessionsPerWeek: arg.SessionsPerWeek}, nil
}

func (f *fakeQueries) DeactivateUserTrainings(_ context.Context, userID uuid.UUID) error {
	f.calls = append(f.calls, "DeactivateUserTrainings")
	f.deactivated = append(f.deactivated, userID)
	return nil
}

func (f *fakeQueries) DeleteTraining(context.Context, repos.DeleteTrainingParams) (int64, error) {
	f.calls = append(f.calls, "DeleteTraining")
	return f.deleteRows, nil
}

func (f *fakeQueries) InsertTrainingUnit(_ context.Context, arg repos.InsertTrainingUnitParams) (uuid.UUID, error) {
	f.calls = append(f.calls, "InsertTrainingUnit")
	f.insertedUnits = append(f.insertedUnits, arg)
	return uuid.New(), nil
}

func (f *fakeQueries) UpdateTrainingUnit(_ context.Context, arg repos.UpdateTrainingUnitParams) (int64, error) {
	f.calls = append(f.calls, "UpdateTrainingUnit")
	if !f.knownUnits[arg.ID] {
		return 0, nil
	}
	f.updatedUnits = append(f.updatedUnits, arg)
	return 1, nil
}

func (f *fakeQueries) DeleteTrainingUnitsExcept(_ context.Context, arg repos.DeleteTrainingUnitsExceptParams) error {
	f.calls = append(f.calls, "DeleteTrainingUnitsExcept")
	f.keptUnits = arg.KeepIds
	return nil
}

func (f *fakeQueries) DeleteTrainingBlocks(context.Context, uuid.UUID) error {
	f.calls = append(f.calls, "DeleteTrainingBlocks")
	return nil
}

func (f *fakeQueries) InsertTrainingBlock(_ context.Context, arg repos.InsertTrainingBlockParams) (uuid.UUID, error) {
	f.calls = append(f.calls, "InsertTrainingBlock")
	f.insertedBlocks = append(f.insertedBlocks, arg)
	return uuid.New(), nil
}

func (f *fakeQueries) InsertTrainingExercise(_ context.Context, arg repos.InsertTrainingExerciseParams) (int64, error) {
	f.calls = append(f.calls, "InsertTrainingExercise")
	if f.insertExerErr != nil {
		return 0, f.insertExerErr
	}
	if f.foreignExer[arg.ExerciseID] {
		return 0, nil
	}
	f.insertedExercises = append(f.insertedExercises, arg)
	return 1, nil
}

func (f *fakeQueries) ListBlockKinds(context.Context) ([]repos.ListBlockKindsRow, error) {
	f.calls = append(f.calls, "ListBlockKinds")
	return f.kinds, nil
}

func (f *fakeQueries) ListRepUnits(context.Context) ([]repos.ListRepUnitsRow, error) {
	f.calls = append(f.calls, "ListRepUnits")
	return f.repUnits, nil
}

func (f *fakeQueries) ListLoadUnits(context.Context) ([]repos.ListLoadUnitsRow, error) {
	f.calls = append(f.calls, "ListLoadUnits")
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
