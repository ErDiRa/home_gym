package training

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// List returns all trainings with their client, for the admin overview.
func (s *Service) List(ctx context.Context) ([]repos.ListTrainingsRow, error) {
	return s.q.ListTrainings(ctx)
}
