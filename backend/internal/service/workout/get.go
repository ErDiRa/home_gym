package workout

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Get returns one of the user's workouts with its sets and scores.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Workout, error) {
	w, err := s.q.GetWorkout(ctx, repos.GetWorkoutParams{ID: id, UserID: userID})
	if err != nil {
		return Workout{}, mapError(err)
	}
	return withLogs(ctx, s.q, w)
}

// List returns the user's workouts finished in [from, to), newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]repos.ListWorkoutsRow, error) {
	fromTime, toTime, err := period(from, to)
	if err != nil {
		return nil, err
	}
	return s.q.ListWorkouts(ctx, repos.ListWorkoutsParams{UserID: userID, FromTime: fromTime, ToTime: toTime})
}
