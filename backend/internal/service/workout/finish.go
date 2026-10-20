package workout

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Finish completes one of the user's workouts with notes on how it went. Finishing a finished
// workout again only replaces its notes.
func (s *Service) Finish(ctx context.Context, userID, id uuid.UUID, notes string) (Workout, error) {
	notes = strings.TrimSpace(notes)
	if utf8.RuneCountInString(notes) > 2000 {
		return Workout{}, service.Invalid("notes can have at most 2000 characters")
	}
	w, err := s.q.FinishWorkout(ctx, repos.FinishWorkoutParams{ID: id, UserID: userID, Notes: notes})
	if err != nil {
		return Workout{}, mapError(err)
	}
	return withLogs(ctx, s.q, w)
}

// Delete soft-deletes one of the user's workouts: discards the one in progress, or removes a
// finished one from the history.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteWorkout(ctx, repos.DeleteWorkoutParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("workout not found", nil)
	}
	return nil
}
