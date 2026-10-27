// Package app holds the whoop module's business logic: connecting a user's WHOOP account and
// linking the user's workouts (the tracking module's) to their WHOOP activities.
//
// The user connects their WHOOP account once (OAuth). On a finished workout they pick one of that
// day's WHOOP activities, whose numbers (strain, heart rate, energy, heart-rate zones) and the
// recovery of that day are fetched from the WHOOP API and stored with the workout.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Tokens are a user's WHOOP OAuth tokens. The refresh token rotates: every refresh returns a new
// one, which replaces the stored one.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// Activity is a WHOOP workout. The numbers are nil while WHOOP hasn't scored it (ScoreState
// PENDING_SCORE).
type Activity struct {
	ID         uuid.UUID
	SportName  string
	Start, End time.Time
	ScoreState string // SCORED, PENDING_SCORE or UNSCORABLE
	Strain     *float64
	AvgHR      *int32
	MaxHR      *int32
	Kilojoule  *float64
	ZoneMs     []int32 // time in heart-rate zones 0 to 5, in milliseconds
}

// Recovery is a WHOOP recovery's score, rounded.
type Recovery struct {
	Score     int32
	RestingHR int32
	HrvMs     float64
}

// Data is the WHOOP activity linked to a workout, with the recovery of that day.
type Data struct {
	WorkoutID uuid.UUID
	Activity
	Recovery  *Recovery
	FetchedAt time.Time
}

// API is the WHOOP API.
type API interface {
	// Configured reports whether the WHOOP app is set up, without which nobody can connect.
	Configured() bool
	// AuthURL is the WHOOP page where the user allows access; WHOOP then redirects to the
	// app's redirect URL with a code and the given state.
	AuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (Tokens, error)
	// Activities lists the user's activities that started in [from, to), newest first.
	Activities(ctx context.Context, accessToken string, from, to time.Time) ([]Activity, error)
	Activity(ctx context.Context, accessToken string, id uuid.UUID) (Activity, error)
	// LatestRecovery returns the newest scored recovery in [from, to), or nil.
	LatestRecovery(ctx context.Context, accessToken string, from, to time.Time) (*Recovery, error)
}

var (
	// ErrRejected is the API's answer when WHOOP no longer accepts the tokens.
	ErrRejected = errors.New("WHOOP rejected the tokens")
	// ErrUnknownActivity is the API's answer for an activity WHOOP doesn't know (or another
	// member's).
	ErrUnknownActivity = errors.New("unknown WHOOP activity")
	// ErrNotConnected is a ConnectionRepository's answer for a user without a connection.
	ErrNotConnected = errors.New("WHOOP is not connected")
)

// ConnectionRepository stores the users' tokens.
type ConnectionRepository interface {
	// Connection returns the user's tokens, or ErrNotConnected.
	Connection(ctx context.Context, userID uuid.UUID) (Tokens, error)
	SaveConnection(ctx context.Context, userID uuid.UUID, tokens Tokens) error
	DeleteConnection(ctx context.Context, userID uuid.UUID) error
}

// DataRepository stores the WHOOP data linked to workouts.
type DataRepository interface {
	// SaveData replaces the data linked to the workout.
	SaveData(ctx context.Context, d Data) (Data, error)
	// Data returns the data linked to the workout, or nil.
	Data(ctx context.Context, workoutID uuid.UUID) (*Data, error)
	// DeleteData unlinks the data, reporting whether there was any.
	DeleteData(ctx context.Context, workoutID uuid.UUID) (bool, error)
}

// Tracking is what the whoop module reads of the tracking module (its contract).
type Tracking interface {
	UserOwnsWorkout(ctx context.Context, userID, workoutID uuid.UUID) (bool, error)
}

// Service implements the WHOOP usecases.
type Service struct {
	api         API
	connections ConnectionRepository
	data        DataRepository
	tracking    Tracking
	now         func() time.Time
}

func NewService(api API, connections ConnectionRepository, data DataRepository, tracking Tracking) *Service {
	if api == nil {
		panic("api cannot be nil")
	}
	if connections == nil {
		panic("connections cannot be nil")
	}
	if data == nil {
		panic("data cannot be nil")
	}
	if tracking == nil {
		panic("tracking cannot be nil")
	}
	return &Service{api: api, connections: connections, data: data, tracking: tracking, now: time.Now}
}
