package cycle

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// fakeQueries is an in-memory Queries: it returns the canned rows set on it and records the calls
// and the parameters of the writes.
type fakeQueries struct {
	calls []string

	// Canned results.
	cycle      *repos.GetCycleRow // nil: no cycle found (GetCycle, UpdateCycle)
	goals      []repos.ListCycleGoalsRow
	trainings  []repos.ListCycleTrainingsRow
	deleteRows int64

	// Recorded writes.
	created            []repos.CreateCycleParams
	updated            []repos.UpdateCycleParams
	insertedGoals      []repos.InsertCycleGoalParams
	clearedGoalsOf     []uuid.UUID
	clearedCycleOf     []uuid.UUID // ClearCycleFromTrainings
	deletedTrainingsOf []uuid.UUID // DeleteCycleTrainings
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

func (f *fakeQueries) called(name string) bool { return f.indexOf(name) >= 0 }

// indexOf returns the position of the first call to name, or -1.
func (f *fakeQueries) indexOf(name string) int {
	for i, c := range f.calls {
		if c == name {
			return i
		}
	}
	return -1
}

// endDate is what the generated column computes: the last Sunday of the weeks.
func endDate(start pgtype.Date, weeks int32) pgtype.Date {
	return pgtype.Date{Time: start.Time.AddDate(0, 0, int(weeks)*7-1), Valid: start.Valid}
}

func (f *fakeQueries) ListCycles(context.Context, uuid.UUID) ([]repos.ListCyclesRow, error) {
	f.calls = append(f.calls, "ListCycles")
	return []repos.ListCyclesRow{}, nil
}

func (f *fakeQueries) GetCycle(_ context.Context, arg repos.GetCycleParams) (repos.GetCycleRow, error) {
	f.calls = append(f.calls, "GetCycle")
	if f.cycle == nil {
		return repos.GetCycleRow{}, pgx.ErrNoRows
	}
	row := *f.cycle
	row.ID = arg.ID
	return row, nil
}

func (f *fakeQueries) ListCycleGoals(context.Context, uuid.UUID) ([]repos.ListCycleGoalsRow, error) {
	f.calls = append(f.calls, "ListCycleGoals")
	return f.goals, nil
}

func (f *fakeQueries) ListCycleTrainings(context.Context, uuid.UUID) ([]repos.ListCycleTrainingsRow, error) {
	f.calls = append(f.calls, "ListCycleTrainings")
	return f.trainings, nil
}

func (f *fakeQueries) CreateCycle(_ context.Context, arg repos.CreateCycleParams) (repos.CreateCycleRow, error) {
	f.calls = append(f.calls, "CreateCycle")
	f.created = append(f.created, arg)
	return repos.CreateCycleRow{ID: uuid.New(), Name: arg.Name, Description: arg.Description,
		StartDate: arg.StartDate, EndDate: endDate(arg.StartDate, arg.Weeks), Weeks: arg.Weeks}, nil
}

func (f *fakeQueries) UpdateCycle(_ context.Context, arg repos.UpdateCycleParams) (repos.UpdateCycleRow, error) {
	f.calls = append(f.calls, "UpdateCycle")
	if f.cycle == nil {
		return repos.UpdateCycleRow{}, pgx.ErrNoRows
	}
	f.updated = append(f.updated, arg)
	return repos.UpdateCycleRow{ID: arg.ID, Name: arg.Name, Description: arg.Description,
		StartDate: arg.StartDate, EndDate: endDate(arg.StartDate, arg.Weeks), Weeks: arg.Weeks}, nil
}

func (f *fakeQueries) DeleteCycle(context.Context, repos.DeleteCycleParams) (int64, error) {
	f.calls = append(f.calls, "DeleteCycle")
	return f.deleteRows, nil
}

func (f *fakeQueries) DeleteCycleGoals(_ context.Context, cycleID uuid.UUID) error {
	f.calls = append(f.calls, "DeleteCycleGoals")
	f.clearedGoalsOf = append(f.clearedGoalsOf, cycleID)
	return nil
}

func (f *fakeQueries) InsertCycleGoal(_ context.Context, arg repos.InsertCycleGoalParams) error {
	f.calls = append(f.calls, "InsertCycleGoal")
	f.insertedGoals = append(f.insertedGoals, arg)
	return nil
}

func (f *fakeQueries) ClearCycleFromTrainings(_ context.Context, cycleID uuid.UUID) error {
	f.calls = append(f.calls, "ClearCycleFromTrainings")
	f.clearedCycleOf = append(f.clearedCycleOf, cycleID)
	return nil
}

func (f *fakeQueries) DeleteCycleTrainings(_ context.Context, cycleID uuid.UUID) error {
	f.calls = append(f.calls, "DeleteCycleTrainings")
	f.deletedTrainingsOf = append(f.deletedTrainingsOf, cycleID)
	return nil
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
