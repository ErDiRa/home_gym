package exercise

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// Update replaces an exercise's fields, links and videos.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in Input) (repos.UpdateExerciseRow, error) {
	if err := validate(&in); err != nil {
		return repos.UpdateExerciseRow{}, err
	}
	exercise, err := s.q.UpdateExercise(ctx, repos.UpdateExerciseParams{
		ID:             id,
		UserID:         userID,
		Name:           in.Name,
		Description:    in.Description,
		Type:           in.Type,
		VideoUrls:      in.VideoURLs,
		JointIds:       in.JointIDs,
		MuscleGroupIds: in.MuscleGroupIDs,
		FocusIds:       in.FocusIDs,
	})
	if err != nil {
		return repos.UpdateExerciseRow{}, mapError(err)
	}
	return exercise, nil
}
