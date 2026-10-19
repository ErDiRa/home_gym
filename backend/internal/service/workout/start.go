package workout

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Start begins a workout of a planned slot of one of the user's trainings. A user does one
// workout at a time: while one is in progress, starting another is a conflict.
func (s *Service) Start(ctx context.Context, userID, weekSessionID uuid.UUID) (Workout, error) {
	slot, err := s.q.GetWeekSession(ctx, repos.GetWeekSessionParams{ID: weekSessionID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Workout{}, service.InvalidCause("unknown planned session", err)
	}
	if err != nil {
		return Workout{}, err
	}
	w, err := s.q.CreateWorkout(ctx, repos.CreateWorkoutParams{
		UserID: userID, TrainingID: &slot.TrainingID, WeekSessionID: &slot.ID, SessionID: &slot.SessionID, SessionName: slot.Name,
	})
	if isUniqueViolation(err) {
		return Workout{}, service.Conflict("finish or discard the workout in progress first", err)
	}
	if err != nil {
		return Workout{}, err
	}
	return fromRow(w, []repos.ListWorkoutSetsRow{}), nil
}
