package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

// StartWorkout begins a workout of a planned slot of one of the user's trainings. Several workouts
// can be in progress at once (e.g. a warm-up and the strength session), but a slot has at most
// one: starting a slot that is already in progress returns that workout instead, with created
// false (e.g. started meanwhile on another device).
func (s *Service) StartWorkout(ctx context.Context, userID, slotID uuid.UUID) (id uuid.UUID, created bool, err error) {
	slot, err := s.planning.Slot(ctx, userID, slotID)
	if errors.Is(err, planning.ErrNotFound) {
		return uuid.Nil, false, errs.InvalidCause("unknown planned session", err)
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	id, err = s.workouts.CreateWorkout(ctx, userID, slot)
	if errors.Is(err, ErrSlotInProgress) {
		id, err = s.workouts.SlotWorkoutInProgress(ctx, userID, slot.ID)
		return id, false, err
	}
	return id, err == nil, err
}

// FinishWorkout completes one of the user's workouts with notes on how it went. completedAt, when
// given, is when it was done (e.g. logged a day later), else now; the start moves along. Finishing
// a finished workout again only replaces its notes, and its completion time when given.
func (s *Service) FinishWorkout(ctx context.Context, userID, id uuid.UUID, notes string, completedAt *time.Time) error {
	notes = strings.TrimSpace(notes)
	if tooLong(notes, 2000) {
		return errs.Invalid("notes can have at most 2000 characters")
	}
	if completedAt != nil {
		now := s.now()
		if completedAt.After(now.Add(5*time.Minute)) || completedAt.Before(now.AddDate(-1, 0, 0)) {
			return errs.Invalid("the workout date must be within the last year and not in the future")
		}
	}
	return workoutError(s.workouts.FinishWorkout(ctx, userID, id, notes, completedAt))
}

// DeleteWorkout soft-deletes one of the user's workouts: discards the one in progress, or removes
// a finished one from the history.
func (s *Service) DeleteWorkout(ctx context.Context, userID, id uuid.UUID) error {
	return workoutError(s.workouts.DeleteWorkout(ctx, userID, id))
}

// CheckPeriod checks a [from, to) range of finished workouts to list: at most a year.
func CheckPeriod(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return errs.Invalid("from and to are required (RFC 3339), from before to")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return errs.Invalid("the period can be at most a year")
	}
	return nil
}

func workoutError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("workout not found", err)
	}
	return err
}

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }
