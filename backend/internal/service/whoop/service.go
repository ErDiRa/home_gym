// Package whoop links the user's workouts to their WHOOP activities.
//
// The user connects their WHOOP account once (OAuth). On a finished workout they pick one of
// that day's WHOOP activities, whose numbers (strain, heart rate, energy, heart-rate zones) and
// the recovery of that day are fetched from the WHOOP API (v2) and stored with the workout.
package whoop

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	GetConnection(ctx context.Context, userID uuid.UUID) (repos.WhoopConnection, error)
	SaveConnection(ctx context.Context, arg repos.SaveConnectionParams) error
	DeleteConnection(ctx context.Context, userID uuid.UUID) error
	UserOwnsWorkout(ctx context.Context, arg repos.UserOwnsWorkoutParams) (bool, error)
	GetWorkoutData(ctx context.Context, arg repos.GetWorkoutDataParams) (repos.WorkoutWhoopDatum, error)
	SaveWorkoutData(ctx context.Context, arg repos.SaveWorkoutDataParams) (repos.WorkoutWhoopDatum, error)
	DeleteWorkoutData(ctx context.Context, arg repos.DeleteWorkoutDataParams) (int64, error)
}

// Config is the app registered in the WHOOP Developer Dashboard.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string // where WHOOP sends the user back to: <origin>/api/whoop/callback
}

// Service implements the WHOOP usecases.
type Service struct {
	q       Queries
	cfg     Config
	client  *http.Client
	baseURL string // the WHOOP API; a test server in tests
}

// New builds the service on Postgres and the WHOOP API.
func New(pool *pgxpool.Pool, cfg Config) *Service {
	return &Service{q: repos.New(pool), cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}, baseURL: "https://api.prod.whoop.com"}
}

// errReconnect is returned when WHOOP no longer accepts the stored tokens.
var errReconnect = service.Unauthorized("WHOOP rejected the connection, connect it again")

// errNotConnected is returned when the user hasn't connected WHOOP.
var errNotConnected = service.Invalid("WHOOP is not connected")

// checkWorkout makes sure the workout is one of the user's.
func (s *Service) checkWorkout(ctx context.Context, userID, workoutID uuid.UUID) error {
	ok, err := s.q.UserOwnsWorkout(ctx, repos.UserOwnsWorkoutParams{ID: workoutID, UserID: userID})
	if err != nil {
		return err
	}
	if !ok {
		return service.NotFound("workout not found", nil)
	}
	return nil
}
