package feedback

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// RatingInput is a user's rating (1-5) of an exercise with optional notes.
type RatingInput struct {
	Rating int32  `json:"rating"`
	Notes  string `json:"notes"`
}

// Rate stores a user's rating and notes for one of their exercises, replacing any earlier
// rating. The exercise picker shows and filters by them.
func (s *Service) Rate(ctx context.Context, userID, exerciseID uuid.UUID, in RatingInput) (repos.ExerciseFeedback, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return repos.ExerciseFeedback{}, service.Invalid("rating must be between 1 and 5")
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if utf8.RuneCountInString(in.Notes) > 1000 {
		return repos.ExerciseFeedback{}, service.Invalid("notes can have at most 1000 characters")
	}
	feedback, err := s.q.UpsertExerciseFeedback(ctx, repos.UpsertExerciseFeedbackParams{
		UserID:     userID,
		ExerciseID: exerciseID,
		Rating:     in.Rating,
		Notes:      in.Notes,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return repos.ExerciseFeedback{}, service.NotFound("exercise not found", err)
	}
	return feedback, err
}

// Ratings returns how the user rated their exercises.
func (s *Service) Ratings(ctx context.Context, userID uuid.UUID) ([]repos.ListExerciseRatingsRow, error) {
	return s.q.ListExerciseRatings(ctx, userID)
}
