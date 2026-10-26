package maxes

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// Delete removes an entry from the user's 1RM history, e.g. one entered by mistake; the entry
// before it becomes the current max again.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteExerciseMax(ctx, repos.DeleteExerciseMaxParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("1RM not found", nil)
	}
	return nil
}
