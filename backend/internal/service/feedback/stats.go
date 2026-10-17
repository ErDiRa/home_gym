package feedback

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// Stats lists the exercises a user has logged, most performed first, with their rating of each.
func (s *Service) Stats(ctx context.Context, userID uuid.UUID) ([]repos.GetExerciseStatsRow, error) {
	return s.q.GetExerciseStats(ctx, userID)
}
