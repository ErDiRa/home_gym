// Package app holds the nutrition module's business logic: a user's body (profile and weight
// history), the macros they ate per day and the daily targets estimated from them.
//
// A day's target is for keeping the weight (maintenance): the energy spent that day (WHOOP's
// measurement when connected, else an estimate from the body and the workouts finished that day)
// split into protein and fat per kg of bodyweight and the rest as carbs.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	tracking "github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

// Sex is what the energy estimate (Mifflin-St Jeor) tells apart.
type Sex string

const (
	Male   Sex = "male"
	Female Sex = "female"
)

// Profile is what the energy estimate needs of a user's body besides the weight.
type Profile struct {
	HeightCm  float64
	BirthDate time.Time
	Sex       Sex
	// ActivityFactor scales the resting energy for the day outside training (1.2 at a desk).
	ActivityFactor float64
}

// Weight is an entry of a user's weight history, with the body fat when it was measured too.
type Weight struct {
	ID             uuid.UUID
	MeasuredOn     time.Time
	WeightKg       float64
	BodyFatPercent *float64
	Notes          string
}

// Macros are the totals a user ate on a day; each may be missing, not all.
type Macros struct {
	Day      time.Time
	ProteinG *float64
	FatG     *float64
	CarbsG   *float64
	Kcal     float64
	Notes    string
	Updated  time.Time
}

// ErrNotFound is a repository's answer for a missing (or another user's) row.
var ErrNotFound = errors.New("not found")

// Repository stores the users' bodies and macros.
type Repository interface {
	// Profile returns the user's profile, or nil.
	Profile(ctx context.Context, userID uuid.UUID) (*Profile, error)
	SaveProfile(ctx context.Context, userID uuid.UUID, p Profile) error
	// Weights lists the user's weights, newest first.
	Weights(ctx context.Context, userID uuid.UUID) ([]Weight, error)
	// SaveWeight replaces the weight of its day.
	SaveWeight(ctx context.Context, userID uuid.UUID, w Weight) (Weight, error)
	// DeleteWeight removes an entry, or returns ErrNotFound.
	DeleteWeight(ctx context.Context, userID, id uuid.UUID) error
	// Macros lists the user's macros of the days in [from, to], oldest first.
	Macros(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Macros, error)
	// SaveMacros replaces the macros of their day.
	SaveMacros(ctx context.Context, userID uuid.UUID, m Macros) (Macros, error)
	// DeleteMacros removes a day's macros, or returns ErrNotFound.
	DeleteMacros(ctx context.Context, userID uuid.UUID, day time.Time) error
}

// Tracking is what the nutrition module reads of the tracking module (its contract).
type Tracking interface {
	FinishedWorkouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]tracking.FinishedWorkout, error)
}

// Whoop is what the nutrition module reads of the whoop module (its contract).
type Whoop interface {
	DailyEnergy(ctx context.Context, userID uuid.UUID, from, to time.Time, loc *time.Location) (map[string]float64, error)
}

// Service implements the nutrition usecases.
type Service struct {
	repo     Repository
	tracking Tracking
	whoop    Whoop
	now      func() time.Time
}

func NewService(repo Repository, tracking Tracking, whoop Whoop) *Service {
	if repo == nil {
		panic("repo cannot be nil")
	}
	if tracking == nil {
		panic("tracking cannot be nil")
	}
	if whoop == nil {
		panic("whoop cannot be nil")
	}
	return &Service{repo: repo, tracking: tracking, whoop: whoop, now: time.Now}
}
