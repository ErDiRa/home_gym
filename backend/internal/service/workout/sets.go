package workout

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// SetInput is one set as it was done. The exercise's place in the unit (block and position
// in the block, from 1) and the set number identify the set: logging it again corrects it.
type SetInput struct {
	ExerciseID       uuid.UUID  `json:"exercise_id"`
	BlockPosition    int32      `json:"block_position"`
	ExercisePosition int32      `json:"exercise_position"`
	SetNumber        int32      `json:"set_number"`
	RepsValue        *float64   `json:"reps_value"`
	RepsUnit         *uuid.UUID `json:"reps_unit"`
	PerSide          bool       `json:"per_side"`
	LoadValue        *float64   `json:"load_value"` // nil without load, e.g. bodyweight
	LoadUnit         *uuid.UUID `json:"load_unit"`
}

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// LogSet stores a set of one of the user's workouts, or corrects it. Sets can be logged until
// the workout is deleted, so a finished one can still be fixed.
func (s *Service) LogSet(ctx context.Context, userID, workoutID uuid.UUID, in SetInput) (repos.WorkoutSet, error) {
	if err := s.checkSet(ctx, in); err != nil {
		return repos.WorkoutSet{}, err
	}
	if _, err := s.q.GetWorkout(ctx, repos.GetWorkoutParams{ID: workoutID, UserID: userID}); err != nil {
		return repos.WorkoutSet{}, mapError(err)
	}
	set, err := s.q.UpsertWorkoutSet(ctx, repos.UpsertWorkoutSetParams{
		WorkoutID: workoutID, ExerciseID: in.ExerciseID, UserID: userID,
		BlockPosition: in.BlockPosition, ExercisePosition: in.ExercisePosition, SetNumber: in.SetNumber,
		RepsValue: *in.RepsValue, RepsUnit: *in.RepsUnit, PerSide: in.PerSide,
		LoadValue: in.LoadValue, LoadUnit: in.LoadUnit,
	})
	// Another user's, a deleted or an unknown exercise.
	if errors.Is(err, pgx.ErrNoRows) {
		return repos.WorkoutSet{}, service.InvalidCause("unknown exercise", err)
	}
	return set, err
}

// DeleteSet removes a set from one of the user's workouts.
func (s *Service) DeleteSet(ctx context.Context, userID, workoutID, setID uuid.UUID) error {
	if _, err := s.q.GetWorkout(ctx, repos.GetWorkoutParams{ID: workoutID, UserID: userID}); err != nil {
		return mapError(err)
	}
	n, err := s.q.DeleteWorkoutSet(ctx, repos.DeleteWorkoutSetParams{ID: setID, WorkoutID: workoutID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("set not found", nil)
	}
	return nil
}

// checkSet validates a set: reps are required, load is optional, and both must lie within
// their unit's bounds (e.g. RPE 1-10), like the planned amounts of a training.
func (s *Service) checkSet(ctx context.Context, in SetInput) error {
	switch {
	case in.ExerciseID == uuid.Nil:
		return service.Invalid("exercise_id is required")
	case in.BlockPosition < 1 || in.ExercisePosition < 1:
		return service.Invalid("block_position and exercise_position must be 1 or more")
	case in.SetNumber < 1 || in.SetNumber > 100:
		return service.Invalid("set_number must be 1-100")
	case in.RepsValue == nil || in.RepsUnit == nil:
		return service.Invalid("reps need a value and a unit")
	case (in.LoadValue == nil) != (in.LoadUnit == nil):
		return service.Invalid("load needs a value and a unit")
	}

	repUnits, err := s.q.ListRepUnits(ctx)
	if err != nil {
		return err
	}
	i := indexOf(repUnits, func(u repos.ListRepUnitsRow) bool { return u.ID == *in.RepsUnit })
	if i < 0 {
		return service.Invalid("unknown reps unit")
	}
	if msg := checkBounds("reps", *in.RepsValue, repUnits[i].Name, repUnits[i].MinValue, repUnits[i].MaxValue); msg != "" {
		return service.Invalid(msg)
	}
	if in.LoadValue == nil {
		return nil
	}

	loadUnits, err := s.q.ListLoadUnits(ctx)
	if err != nil {
		return err
	}
	i = indexOf(loadUnits, func(u repos.ListLoadUnitsRow) bool { return u.ID == *in.LoadUnit })
	if i < 0 {
		return service.Invalid("unknown load unit")
	}
	if msg := checkBounds("load", *in.LoadValue, loadUnits[i].Name, loadUnits[i].MinValue, loadUnits[i].MaxValue); msg != "" {
		return service.Invalid(msg)
	}
	return nil
}

// checkBounds checks a value against the column's range and its unit's own bounds, returning a
// message when it is out of them.
func checkBounds(what string, v float64, unit string, min, max *float64) string {
	lo, hi := 0.0, float64(maxAmount-1)
	if min != nil {
		lo = *min
	}
	if max != nil {
		hi = *max
	}
	if v < lo || v > hi {
		return fmt.Sprintf("%s in %s must be between %g and %g", what, unit, lo, hi)
	}
	return ""
}

// indexOf returns the index of the first element of s that matches, or -1.
func indexOf[T any](s []T, match func(T) bool) int {
	for i, v := range s {
		if match(v) {
			return i
		}
	}
	return -1
}
