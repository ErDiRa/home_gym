package workout

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Finish completes one of the user's workouts with notes on how it went. completedAt, when
// given, is when it was done (e.g. logged a day later), else now; the start moves along. Finishing
// a finished workout again only replaces its notes, and its completion time when given.
func (s *Service) Finish(ctx context.Context, userID, id uuid.UUID, notes string, completedAt *time.Time) (Workout, error) {
	notes = strings.TrimSpace(notes)
	if utf8.RuneCountInString(notes) > 2000 {
		return Workout{}, service.Invalid("notes can have at most 2000 characters")
	}
	var done pgtype.Timestamptz
	if completedAt != nil {
		if completedAt.After(time.Now().Add(5*time.Minute)) || completedAt.Before(time.Now().AddDate(-1, 0, 0)) {
			return Workout{}, service.Invalid("the workout date must be within the last year and not in the future")
		}
		done = pgtype.Timestamptz{Time: *completedAt, Valid: true}
	}
	w, err := s.q.FinishWorkout(ctx, repos.FinishWorkoutParams{ID: id, UserID: userID, Notes: notes, CompletedAt: done})
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
