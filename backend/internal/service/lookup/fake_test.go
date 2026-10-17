package lookup

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

// fakeQueries is an in-memory Queries; the creates fail with err when it is set.
type fakeQueries struct {
	err error

	focuses   []repos.CreateFocusParams
	repUnits  []repos.CreateRepUnitParams
	loadUnits []repos.CreateLoadUnitParams
}

func (f *fakeQueries) ListJoints(context.Context) ([]repos.ListJointsRow, error) {
	return []repos.ListJointsRow{}, nil
}

func (f *fakeQueries) ListMuscleGroups(context.Context) ([]repos.ListMuscleGroupsRow, error) {
	return []repos.ListMuscleGroupsRow{}, nil
}

func (f *fakeQueries) ListFocuses(context.Context) ([]repos.ListFocusesRow, error) {
	return []repos.ListFocusesRow{}, nil
}

func (f *fakeQueries) CreateFocus(_ context.Context, arg repos.CreateFocusParams) (repos.CreateFocusRow, error) {
	if f.err != nil {
		return repos.CreateFocusRow{}, f.err
	}
	f.focuses = append(f.focuses, arg)
	return repos.CreateFocusRow{ID: uuid.New(), Name: arg.Name, Alias: arg.Alias}, nil
}

func (f *fakeQueries) ListBlockKinds(context.Context) ([]repos.ListBlockKindsRow, error) {
	return []repos.ListBlockKindsRow{}, nil
}

func (f *fakeQueries) ListRepUnits(context.Context) ([]repos.ListRepUnitsRow, error) {
	return []repos.ListRepUnitsRow{}, nil
}

func (f *fakeQueries) CreateRepUnit(_ context.Context, arg repos.CreateRepUnitParams) (repos.CreateRepUnitRow, error) {
	if f.err != nil {
		return repos.CreateRepUnitRow{}, f.err
	}
	f.repUnits = append(f.repUnits, arg)
	return repos.CreateRepUnitRow{ID: uuid.New(), Name: arg.Name, MinValue: arg.MinValue, MaxValue: arg.MaxValue}, nil
}

func (f *fakeQueries) ListLoadUnits(context.Context) ([]repos.ListLoadUnitsRow, error) {
	return []repos.ListLoadUnitsRow{}, nil
}

func (f *fakeQueries) CreateLoadUnit(_ context.Context, arg repos.CreateLoadUnitParams) (repos.CreateLoadUnitRow, error) {
	if f.err != nil {
		return repos.CreateLoadUnitRow{}, f.err
	}
	f.loadUnits = append(f.loadUnits, arg)
	return repos.CreateLoadUnitRow{ID: uuid.New(), Name: arg.Name, MinValue: arg.MinValue, MaxValue: arg.MaxValue}, nil
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

func ptr[T any](v T) *T { return &v }
