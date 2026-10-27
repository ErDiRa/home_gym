package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// RatingInput is a user's rating (1-5) of an exercise with optional notes.
type RatingInput struct {
	Rating int32
	Notes  string
}

// RateExercise stores a user's rating and notes for one of their exercises, replacing any earlier
// rating. The exercise picker shows and filters by them.
func (s *Service) RateExercise(ctx context.Context, userID, exerciseID uuid.UUID, in RatingInput) (Rating, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return Rating{}, errs.Invalid("rating must be between 1 and 5")
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if tooLong(in.Notes, 1000) {
		return Rating{}, errs.Invalid("notes can have at most 1000 characters")
	}
	r, err := s.ratings.RateExercise(ctx, userID, exerciseID, in)
	if errors.Is(err, ErrNotFound) {
		return Rating{}, errs.NotFound("exercise not found", err)
	}
	return r, err
}
