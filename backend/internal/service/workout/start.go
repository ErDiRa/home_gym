package workout

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Start begins a workout of a unit of one of the user's trainings. A user does one workout at
// a time: while one is in progress, starting another is a conflict.
func (s *Service) Start(ctx context.Context, userID, unitID uuid.UUID) (Workout, error) {
	unit, err := s.q.GetUnit(ctx, repos.GetUnitParams{ID: unitID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Workout{}, service.InvalidCause("unknown unit", err)
	}
	if err != nil {
		return Workout{}, err
	}
	w, err := s.q.CreateWorkout(ctx, repos.CreateWorkoutParams{
		UserID: userID, TrainingID: &unit.TrainingID, UnitID: &unit.ID, UnitName: unit.Name,
	})
	if isUniqueViolation(err) {
		return Workout{}, service.Conflict("finish or discard the workout in progress first", err)
	}
	if err != nil {
		return Workout{}, err
	}
	return Workout{
		ID: w.ID, TrainingID: w.TrainingID, UnitID: w.UnitID, UnitName: w.UnitName,
		StartedAt: w.StartedAt, CompletedAt: w.CompletedAt, Notes: w.Notes, Sets: []repos.ListWorkoutSetsRow{},
	}, nil
}
