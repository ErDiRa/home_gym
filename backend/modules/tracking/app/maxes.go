package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// MaxInput is a 1RM: the load in a weight unit, the day it was lifted and, when it was saved from
// a logged single, that set.
type MaxInput struct {
	LoadValue    float64
	LoadUnit     uuid.UUID
	AchievedOn   string // YYYY-MM-DD
	WorkoutSetID *uuid.UUID
	Notes        string
}

// AddMax adds a 1RM to the history of one of the user's exercises; being the newest, it becomes
// the current max (from which the tracker turns loads planned in % 1RM into a weight). Earlier
// entries stay, so the history shows the progress.
func (s *Service) AddMax(ctx context.Context, userID, exerciseID uuid.UUID, in MaxInput) (uuid.UUID, error) {
	day, err := s.checkMax(ctx, userID, exerciseID, &in)
	if err != nil {
		return uuid.Nil, err
	}
	exercises, err := s.catalog.Exercises(ctx, userID, []uuid.UUID{exerciseID})
	if err != nil {
		return uuid.Nil, err
	}
	if e, ok := exercises[exerciseID]; !ok || e.Deleted {
		return uuid.Nil, errs.NotFound("exercise not found", nil)
	}
	return s.maxes.CreateMax(ctx, userID, exerciseID, in, day)
}

// DeleteMax removes an entry from the user's 1RM history, e.g. one entered by mistake; the entry
// before it becomes the current max again.
func (s *Service) DeleteMax(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.maxes.DeleteMax(ctx, userID, id); errors.Is(err, ErrNotFound) {
		return errs.NotFound("1RM not found", err)
	} else if err != nil {
		return err
	}
	return nil
}

// checkMax validates a 1RM, trimming its notes in place, and returns its day.
func (s *Service) checkMax(ctx context.Context, userID, exerciseID uuid.UUID, in *MaxInput) (time.Time, error) {
	if in.LoadValue <= 0 || in.LoadValue >= maxAmount {
		return time.Time{}, errs.Invalid("load must be between 0 and 99999")
	}
	day, err := time.Parse(time.DateOnly, in.AchievedOn)
	if err != nil {
		return time.Time{}, errs.Invalid("achieved_on must be a date (YYYY-MM-DD)")
	}
	// The client sends its local day, which may be a day ahead of the server's.
	if day.Year() < 1900 || day.After(s.now().AddDate(0, 0, 1)) {
		return time.Time{}, errs.Invalid("achieved_on must not be in the future")
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if tooLong(in.Notes, 1000) {
		return time.Time{}, errs.Invalid("notes can have at most 1000 characters")
	}

	units, err := s.catalog.Units(ctx)
	if err != nil {
		return time.Time{}, err
	}
	unit, ok := liveUnit(units.Load, in.LoadUnit)
	if !ok {
		return time.Time{}, errs.Invalid("unknown load unit")
	}
	if unit.KgPerUnit == nil {
		return time.Time{}, errs.Invalid("a 1RM needs a weight unit such as kg or lb")
	}

	if in.WorkoutSetID != nil {
		owns, err := s.maxes.UserOwnsSet(ctx, userID, *in.WorkoutSetID, exerciseID)
		if err != nil {
			return time.Time{}, err
		}
		if !owns {
			return time.Time{}, errs.Invalid("unknown set")
		}
	}
	return day, nil
}
