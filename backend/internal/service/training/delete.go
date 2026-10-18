package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Delete soft-deletes one of the user's trainings.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteTraining(ctx, repos.DeleteTrainingParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("training not found", nil)
	}
	return nil
}
