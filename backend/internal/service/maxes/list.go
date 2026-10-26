package maxes

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// List returns the user's 1RM history of all their exercises, newest first per exercise.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]repos.ListExerciseMaxesRow, error) {
	return s.q.ListExerciseMaxes(ctx, userID)
}
