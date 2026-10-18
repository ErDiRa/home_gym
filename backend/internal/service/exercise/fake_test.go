package exercise

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// fakeQueries is an in-memory Queries returning canned results and recording writes.
type fakeQueries struct {
	err error // returned by every method

	created []repos.CreateExerciseParams
	updated []repos.UpdateExerciseParams
}

func (f *fakeQueries) ListExercises(context.Context, uuid.UUID) ([]repos.ListExercisesRow, error) {
	return []repos.ListExercisesRow{}, f.err
}

func (f *fakeQueries) GetExercise(_ context.Context, arg repos.GetExerciseParams) (repos.GetExerciseRow, error) {
	return repos.GetExerciseRow{ID: arg.ID}, f.err
}

func (f *fakeQueries) CreateExercise(_ context.Context, arg repos.CreateExerciseParams) (repos.CreateExerciseRow, error) {
	if f.err != nil {
		return repos.CreateExerciseRow{}, f.err
	}
	f.created = append(f.created, arg)
	return repos.CreateExerciseRow{ID: uuid.New(), Name: arg.Name, Type: arg.Type, VideoUrls: arg.VideoUrls}, nil
}

func (f *fakeQueries) UpdateExercise(_ context.Context, arg repos.UpdateExerciseParams) (repos.UpdateExerciseRow, error) {
	if f.err != nil {
		return repos.UpdateExerciseRow{}, f.err
	}
	f.updated = append(f.updated, arg)
	return repos.UpdateExerciseRow{ID: arg.ID, Name: arg.Name, Type: arg.Type, VideoUrls: arg.VideoUrls}, nil
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
