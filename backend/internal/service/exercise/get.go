package exercise

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// Get returns one exercise with the ids of its joints, muscle groups and focuses, for filling
// the exercise form.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (repos.GetExerciseRow, error) {
	exercise, err := s.q.GetExercise(ctx, id)
	if err != nil {
		return repos.GetExerciseRow{}, mapError(err)
	}
	return exercise, nil
}
