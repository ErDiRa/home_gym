// Package client is the tracking module's contract: what other modules may read of the workouts.
package client

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Tracking interface {
	// UserOwnsWorkout reports whether the workout is one of the user's live workouts.
	UserOwnsWorkout(ctx context.Context, userID, workoutID uuid.UUID) (bool, error)
	// FinishedWorkouts lists the user's workouts finished in [from, to), oldest first.
	FinishedWorkouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]FinishedWorkout, error)
}

// FinishedWorkout is a workout the user finished: the session done (its name as it was) and when.
type FinishedWorkout struct {
	ID          uuid.UUID
	SessionName string
	StartedAt   time.Time
	CompletedAt time.Time
}
