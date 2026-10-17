package client

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// ExerciseRatings returns how a client rated exercises, so admins can filter by it when
// picking exercises for the client's training. An unknown or deleted client gets none.
func (s *Service) ExerciseRatings(ctx context.Context, id uuid.UUID) ([]repos.ListClientExerciseRatingsRow, error) {
	return s.q.ListClientExerciseRatings(ctx, id)
}
