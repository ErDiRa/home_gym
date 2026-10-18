package workout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Get returns one of the user's workouts with its sets.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Workout, error) {
	w, err := s.q.GetWorkout(ctx, repos.GetWorkoutParams{ID: id, UserID: userID})
	if err != nil {
		return Workout{}, mapError(err)
	}
	return withSets(ctx, s.q, w)
}

// Current returns the user's workout in progress with its sets; nil when there is none.
func (s *Service) Current(ctx context.Context, userID uuid.UUID) (*Workout, error) {
	w, err := s.q.GetCurrentWorkout(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	workout, err := withSets(ctx, s.q, w)
	if err != nil {
		return nil, err
	}
	return &workout, nil
}

// List returns the user's workouts finished in [from, to), newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]repos.ListWorkoutsRow, error) {
	fromTime, toTime, err := period(from, to)
	if err != nil {
		return nil, err
	}
	return s.q.ListWorkouts(ctx, repos.ListWorkoutsParams{UserID: userID, FromTime: fromTime, ToTime: toTime})
}
