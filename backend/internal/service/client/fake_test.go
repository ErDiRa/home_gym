package client

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// fakeQueries is an in-memory Queries returning canned results and recording writes.
type fakeQueries struct {
	adminEmails map[string]bool
	user        repos.GetUserByIDRow
	userErr     error
	writeErr    error // returned by CreateUser and UpdateUser
	deleteRows  int64

	created []repos.CreateUserParams
	updated []repos.UpdateUserParams
}

func newTestService(f *fakeQueries) *Service { return &Service{q: f, hashCost: 4} } // bcrypt.MinCost

func (f *fakeQueries) ListUsers(context.Context) ([]repos.ListUsersRow, error) {
	return []repos.ListUsersRow{}, nil
}

func (f *fakeQueries) CreateUser(_ context.Context, arg repos.CreateUserParams) (repos.CreateUserRow, error) {
	if f.writeErr != nil {
		return repos.CreateUserRow{}, f.writeErr
	}
	f.created = append(f.created, arg)
	return repos.CreateUserRow{ID: uuid.New(), Email: arg.Email, FirstName: arg.FirstName, LastName: arg.LastName}, nil
}

func (f *fakeQueries) GetUserByID(context.Context, uuid.UUID) (repos.GetUserByIDRow, error) {
	return f.user, f.userErr
}

func (f *fakeQueries) UpdateUser(_ context.Context, arg repos.UpdateUserParams) (repos.UpdateUserRow, error) {
	if f.writeErr != nil {
		return repos.UpdateUserRow{}, f.writeErr
	}
	f.updated = append(f.updated, arg)
	return repos.UpdateUserRow{ID: arg.ID, Email: arg.Email, FirstName: arg.FirstName, LastName: arg.LastName}, nil
}

func (f *fakeQueries) DeleteUser(context.Context, uuid.UUID) (int64, error) { return f.deleteRows, nil }

func (f *fakeQueries) ListClientTrainings(context.Context, uuid.UUID) ([]repos.ListClientTrainingsRow, error) {
	return []repos.ListClientTrainingsRow{{Name: "Strength", Active: true}}, nil
}

func (f *fakeQueries) ListClientExerciseRatings(context.Context, uuid.UUID) ([]repos.ListClientExerciseRatingsRow, error) {
	return []repos.ListClientExerciseRatingsRow{}, nil
}

func (f *fakeQueries) AdminEmailExists(_ context.Context, email string) (bool, error) {
	return f.adminEmails[email], nil
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
