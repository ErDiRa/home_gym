package exercise

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// List returns all exercises with the names of their joints, muscle groups and focuses, which
// the lists and the exercise picker search and filter on.
func (s *Service) List(ctx context.Context) ([]repos.ListExercisesRow, error) {
	return s.q.ListExercises(ctx)
}
