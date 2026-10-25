package cycle

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// List returns the user's cycles with their goal and training counts, the latest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]repos.ListCyclesRow, error) {
	return s.q.ListCycles(ctx, userID)
}
