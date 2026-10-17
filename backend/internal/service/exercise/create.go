package exercise

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// Create adds an exercise with its links and videos.
func (s *Service) Create(ctx context.Context, in Input) (repos.CreateExerciseRow, error) {
	if err := validate(&in); err != nil {
		return repos.CreateExerciseRow{}, err
	}
	exercise, err := s.q.CreateExercise(ctx, repos.CreateExerciseParams{
		Name:           in.Name,
		Description:    in.Description,
		Type:           in.Type,
		VideoUrls:      in.VideoURLs,
		JointIds:       in.JointIDs,
		MuscleGroupIds: in.MuscleGroupIDs,
		FocusIds:       in.FocusIDs,
	})
	if err != nil {
		return repos.CreateExerciseRow{}, mapError(err)
	}
	return exercise, nil
}
