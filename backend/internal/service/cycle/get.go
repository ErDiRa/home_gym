package cycle

import (
	"context"

	"github.com/google/uuid"
)

// Get returns one of the user's cycles with its goals and trainings.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Cycle, error) {
	c, err := load(ctx, s.q, userID, id)
	return c, mapError(err)
}
