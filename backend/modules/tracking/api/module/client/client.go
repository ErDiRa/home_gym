// Package client is the tracking module's contract: what other modules may read of the workouts.
package client

import (
	"context"

	"github.com/google/uuid"
)

type Tracking interface {
	// UserOwnsWorkout reports whether the workout is one of the user's live workouts.
	UserOwnsWorkout(ctx context.Context, userID, workoutID uuid.UUID) (bool, error)
}
