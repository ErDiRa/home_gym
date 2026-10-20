package workout

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Start begins a workout of a planned slot of one of the user's trainings. Several workouts can
// be in progress at once (e.g. a warm-up and the strength session), but a slot has at most one:
// starting a slot that is already in progress returns that workout with its sets instead, with
// created false (e.g. started meanwhile on another device).
func (s *Service) Start(ctx context.Context, userID, weekSessionID uuid.UUID) (w Workout, created bool, err error) {
	slot, err := s.q.GetWeekSession(ctx, repos.GetWeekSessionParams{ID: weekSessionID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Workout{}, false, service.InvalidCause("unknown planned session", err)
	}
	if err != nil {
		return Workout{}, false, err
	}
	row, err := s.q.CreateWorkout(ctx, repos.CreateWorkoutParams{
		UserID: userID, TrainingID: &slot.TrainingID, WeekSessionID: &slot.ID, SessionID: &slot.SessionID, SessionName: slot.Name,
	})
	if isUniqueViolation(err) {
		existing, err := s.q.GetSlotWorkoutInProgress(ctx, repos.GetSlotWorkoutInProgressParams{WeekSessionID: &slot.ID, UserID: userID})
		if err != nil {
			return Workout{}, false, err
		}
		w, err := withLogs(ctx, s.q, existing)
		return w, false, err
	}
	if err != nil {
		return Workout{}, false, err
	}
	return fromRow(row, []repos.ListWorkoutSetsRow{}, []repos.WorkoutResult{}), true, nil
}
