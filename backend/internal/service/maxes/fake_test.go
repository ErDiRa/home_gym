package maxes

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// fakeQueries is an in-memory Queries: units by id (kg_per_unit nil = not a weight unit), the
// sets each user owns, and the maxes created. createErr fails the insert.
type fakeQueries struct {
	units     map[uuid.UUID]*float64
	sets      map[uuid.UUID]repos.UserOwnsSetParams // by set id: whose and for which exercise
	createErr error
	created   []repos.CreateExerciseMaxParams
	deleted   []repos.DeleteExerciseMaxParams
	live      map[uuid.UUID]uuid.UUID // max id -> user id
	listed    []uuid.UUID
}

func (f *fakeQueries) ListExerciseMaxes(_ context.Context, userID uuid.UUID) ([]repos.ListExerciseMaxesRow, error) {
	f.listed = append(f.listed, userID)
	return []repos.ListExerciseMaxesRow{}, nil
}

func (f *fakeQueries) GetLoadUnit(_ context.Context, id uuid.UUID) (repos.GetLoadUnitRow, error) {
	kg, ok := f.units[id]
	if !ok {
		return repos.GetLoadUnitRow{}, pgx.ErrNoRows
	}
	return repos.GetLoadUnitRow{Name: "unit", KgPerUnit: kg}, nil
}

func (f *fakeQueries) UserOwnsSet(_ context.Context, arg repos.UserOwnsSetParams) (bool, error) {
	owner, ok := f.sets[arg.ID]
	return ok && owner.UserID == arg.UserID && owner.ExerciseID == arg.ExerciseID, nil
}

func (f *fakeQueries) CreateExerciseMax(_ context.Context, arg repos.CreateExerciseMaxParams) (repos.CreateExerciseMaxRow, error) {
	if f.createErr != nil {
		return repos.CreateExerciseMaxRow{}, f.createErr
	}
	f.created = append(f.created, arg)
	return repos.CreateExerciseMaxRow{
		ID: uuid.New(), ExerciseID: arg.ExerciseID, LoadValue: arg.LoadValue, LoadUnit: arg.LoadUnit,
		AchievedOn: arg.AchievedOn, WorkoutSetID: arg.WorkoutSetID, Notes: arg.Notes,
	}, nil
}

func (f *fakeQueries) DeleteExerciseMax(_ context.Context, arg repos.DeleteExerciseMaxParams) (int64, error) {
	if user, ok := f.live[arg.ID]; !ok || user != arg.UserID {
		return 0, nil
	}
	delete(f.live, arg.ID)
	f.deleted = append(f.deleted, arg)
	return 1, nil
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
