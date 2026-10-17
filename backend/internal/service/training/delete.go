package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

// Delete soft-deletes a training.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteTraining(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("training not found", nil)
	}
	return nil
}
