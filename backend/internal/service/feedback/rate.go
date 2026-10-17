package feedback

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// RatingInput is a client's rating (1-5) of an exercise with optional notes.
type RatingInput struct {
	Rating int32  `json:"rating"`
	Notes  string `json:"notes"`
}

// Rate stores a user's rating and notes for an exercise, replacing any earlier rating of
// theirs. Admins read these per client to filter the exercise picker.
func (s *Service) Rate(ctx context.Context, userID, exerciseID uuid.UUID, in RatingInput) (repos.ExerciseFeedback, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return repos.ExerciseFeedback{}, service.Invalid("rating must be between 1 and 5")
	}
	feedback, err := s.q.UpsertExerciseFeedback(ctx, repos.UpsertExerciseFeedbackParams{
		UserID:     userID,
		ExerciseID: exerciseID,
		Rating:     in.Rating,
		Notes:      in.Notes,
	})
	if isForeignKeyViolation(err) {
		return repos.ExerciseFeedback{}, service.NotFound("user or exercise not found", err)
	}
	return feedback, err
}
