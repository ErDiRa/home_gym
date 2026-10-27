package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
)

// SetInput is one set as it was done. The exercise's place in the session (block and position in
// the block, from 1) and the set number identify the set: logging it again corrects it.
type SetInput struct {
	ExerciseID       uuid.UUID
	BlockPosition    int32
	ExercisePosition int32
	SetNumber        int32
	RepsValue        *float64 // nil when only the load was logged
	RepsUnit         *uuid.UUID
	PerSide          bool
	LoadValue        *float64 // nil without load, e.g. bodyweight
	LoadUnit         *uuid.UUID
}

// LogSet stores a set of one of the user's workouts, or corrects it. Sets can be logged until the
// workout is deleted, so a finished one can still be fixed. The exercise must be one of the user's
// live exercises.
func (s *Service) LogSet(ctx context.Context, userID, workoutID uuid.UUID, in SetInput) (LoggedSet, error) {
	if err := s.checkSet(ctx, in); err != nil {
		return LoggedSet{}, err
	}
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return LoggedSet{}, err
	}
	exercises, err := s.catalog.Exercises(ctx, userID, []uuid.UUID{in.ExerciseID})
	if err != nil {
		return LoggedSet{}, err
	}
	if e, ok := exercises[in.ExerciseID]; !ok || e.Deleted {
		return LoggedSet{}, errs.Invalid("unknown exercise")
	}
	return s.workouts.LogSet(ctx, workoutID, in)
}

// DeleteSet removes a set from one of the user's workouts.
func (s *Service) DeleteSet(ctx context.Context, userID, workoutID, setID uuid.UUID) error {
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	if err := s.workouts.DeleteSet(ctx, workoutID, setID); errors.Is(err, ErrNotFound) {
		return errs.NotFound("set not found", err)
	} else if err != nil {
		return err
	}
	return nil
}

// checkWorkout fails with "workout not found" unless the workout is one of the user's live ones.
func (s *Service) checkWorkout(ctx context.Context, userID, workoutID uuid.UUID) error {
	ok, err := s.workouts.UserOwnsWorkout(ctx, userID, workoutID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.NotFound("workout not found", nil)
	}
	return nil
}

// checkSet validates a set: whatever was entered of it is kept, so reps and load are each
// optional but not both; the reps unit is always given (the plan's). Values must lie within their
// unit's bounds (e.g. RPE 1-10), like the planned amounts of a training.
func (s *Service) checkSet(ctx context.Context, in SetInput) error {
	switch {
	case in.ExerciseID == uuid.Nil:
		return errs.Invalid("exercise_id is required")
	case in.BlockPosition < 1 || in.ExercisePosition < 1:
		return errs.Invalid("block_position and exercise_position must be 1 or more")
	case in.SetNumber < 1 || in.SetNumber > 100:
		return errs.Invalid("set_number must be 1-100")
	case in.RepsUnit == nil:
		return errs.Invalid("reps_unit is required")
	case (in.LoadValue == nil) != (in.LoadUnit == nil):
		return errs.Invalid("load needs a value and a unit")
	case in.RepsValue == nil && in.LoadValue == nil:
		return errs.Invalid("enter the reps or the load")
	}

	units, err := s.catalog.Units(ctx)
	if err != nil {
		return err
	}
	repUnit, ok := liveUnit(units.Rep, *in.RepsUnit)
	if !ok {
		return errs.Invalid("unknown reps unit")
	}
	if in.RepsValue != nil {
		if msg := checkBounds("reps", *in.RepsValue, repUnit); msg != "" {
			return errs.Invalid(msg)
		}
	}
	if in.LoadValue == nil {
		return nil
	}
	loadUnit, ok := liveUnit(units.Load, *in.LoadUnit)
	if !ok {
		return errs.Invalid("unknown load unit")
	}
	if msg := checkBounds("load", *in.LoadValue, loadUnit); msg != "" {
		return errs.Invalid(msg)
	}
	return nil
}

// liveUnit finds a live unit by id.
func liveUnit(units []catalog.Unit, id uuid.UUID) (catalog.Unit, bool) {
	for _, u := range units {
		if u.ID == id && !u.Deleted {
			return u, true
		}
	}
	return catalog.Unit{}, false
}

// checkBounds checks a value against the column's range and its unit's own bounds, returning a
// message when it is out of them.
func checkBounds(what string, v float64, unit catalog.Unit) string {
	lo, hi := 0.0, float64(maxAmount-1)
	if unit.MinValue != nil {
		lo = *unit.MinValue
	}
	if unit.MaxValue != nil {
		hi = *unit.MaxValue
	}
	if v < lo || v > hi {
		return fmt.Sprintf("%s in %s must be between %g and %g", what, unit.Name, lo, hi)
	}
	return ""
}
