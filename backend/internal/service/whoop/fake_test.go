package whoop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

// fakeQueries is an in-memory Queries for one user: the connection, which workouts are theirs
// and the data saved.
type fakeQueries struct {
	conn     *repos.WhoopConnection
	workouts map[uuid.UUID]bool // the user's workouts
	saved    []repos.SaveWorkoutDataParams
}

func (f *fakeQueries) GetConnection(context.Context, uuid.UUID) (repos.WhoopConnection, error) {
	if f.conn == nil {
		return repos.WhoopConnection{}, pgx.ErrNoRows
	}
	return *f.conn, nil
}

func (f *fakeQueries) SaveConnection(_ context.Context, arg repos.SaveConnectionParams) error {
	f.conn = &repos.WhoopConnection{UserID: arg.UserID, AccessToken: arg.AccessToken, RefreshToken: arg.RefreshToken, ExpiresAt: arg.ExpiresAt}
	return nil
}

func (f *fakeQueries) DeleteConnection(context.Context, uuid.UUID) error {
	f.conn = nil
	return nil
}

func (f *fakeQueries) UserOwnsWorkout(_ context.Context, arg repos.UserOwnsWorkoutParams) (bool, error) {
	return f.workouts[arg.ID], nil
}

func (f *fakeQueries) GetWorkoutData(context.Context, repos.GetWorkoutDataParams) (repos.WorkoutWhoopDatum, error) {
	return repos.WorkoutWhoopDatum{}, pgx.ErrNoRows
}

func (f *fakeQueries) SaveWorkoutData(_ context.Context, arg repos.SaveWorkoutDataParams) (repos.WorkoutWhoopDatum, error) {
	f.saved = append(f.saved, arg)
	return repos.WorkoutWhoopDatum{WorkoutID: arg.WorkoutID, WhoopWorkoutID: arg.WhoopWorkoutID, Strain: arg.Strain}, nil
}

func (f *fakeQueries) DeleteWorkoutData(context.Context, repos.DeleteWorkoutDataParams) (int64, error) {
	return 0, nil
}

// newTestService runs the service against a fake WHOOP API answering with the handlers in
// routes (by path).
func newTestService(t *testing.T, f *fakeQueries, routes map[string]http.HandlerFunc) *Service {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range routes {
		mux.HandleFunc(path, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Service{q: f, cfg: Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, client: srv.Client(), baseURL: srv.URL}
}
