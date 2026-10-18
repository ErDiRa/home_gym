package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// List returns the user's trainings with their size, for the overview.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]repos.ListTrainingsRow, error) {
	return s.q.ListTrainings(ctx, userID)
}
