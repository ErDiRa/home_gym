package feedback

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// fakeQueries is an in-memory Queries; the upsert fails with err when it is set.
type fakeQueries struct {
	err     error
	upserts []repos.UpsertExerciseFeedbackParams
}

func (f *fakeQueries) UpsertExerciseFeedback(_ context.Context, arg repos.UpsertExerciseFeedbackParams) (repos.ExerciseFeedback, error) {
	if f.err != nil {
		return repos.ExerciseFeedback{}, f.err
	}
	f.upserts = append(f.upserts, arg)
	return repos.ExerciseFeedback{ID: uuid.New(), UserID: arg.UserID, ExerciseID: arg.ExerciseID, Rating: arg.Rating, Notes: arg.Notes}, nil
}

func (f *fakeQueries) GetExerciseStats(context.Context, uuid.UUID) ([]repos.GetExerciseStatsRow, error) {
	return []repos.GetExerciseStatsRow{}, nil
}

func TestRate(t *testing.T) {
	f := &fakeQueries{}
	user, exercise := uuid.New(), uuid.New()
	got, err := (&Service{q: f}).Rate(context.Background(), user, exercise, RatingInput{Rating: 4, Notes: "knees ok"})
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != user || got.ExerciseID != exercise || got.Rating != 4 || got.Notes != "knees ok" {
		t.Errorf("feedback = %+v", got)
	}
}

func TestRateRejects(t *testing.T) {
	for _, rating := range []int32{0, 6, -1} {
		f := &fakeQueries{}
		_, err := (&Service{q: f}).Rate(context.Background(), uuid.New(), uuid.New(), RatingInput{Rating: rating})
		wantError(t, err, service.KindInvalid, "rating must be between 1 and 5")
		if len(f.upserts) != 0 {
			t.Errorf("rating %d was stored", rating)
		}
	}

	f := &fakeQueries{err: &pgconn.PgError{Code: "23503"}}
	_, err := (&Service{q: f}).Rate(context.Background(), uuid.New(), uuid.New(), RatingInput{Rating: 3})
	wantError(t, err, service.KindNotFound, "user or exercise not found")
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
