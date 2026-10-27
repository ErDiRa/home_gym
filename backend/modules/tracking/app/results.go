package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// The formats of a WOD block's score, as in the tracking.workout_results.score_type CHECK (and the
// planning module's Score* constants).
const (
	scoreForTime     = "for_time"
	scoreAMRAP       = "amrap"
	scoreMaxReps     = "max_reps"
	scoreMaxCalories = "max_calories"
	scoreEMOM        = "emom"
)

// ResultInput is the score of a WOD block as it was done. The block's place in the session
// (from 1) identifies it: logging it again corrects it. ScoreType is the block's format as
// planned, and it decides which of the values count:
//
//	for_time      TimeSeconds, or Reps when the time cap was hit
//	amrap         Rounds, plus the Reps of the unfinished round (optional)
//	max_reps      Reps
//	max_calories  Calories
//	emom          Rounds: the intervals completed
//
// The others are dropped. Rx is false when the WOD was scaled.
type ResultInput struct {
	BlockPosition int32
	ScoreType     string
	TimeSeconds   *int32
	Rounds        *int32
	Reps          *int32
	Calories      *int32
	Rx            bool
	Notes         string
}

// Bounds of a score, as in the tracking.workout_results CHECKs.
const (
	maxResultTime   = 24 * 60 * 60
	maxResultRounds = 10000
	maxResultCount  = 100000 // reps and calories
)

// LogResult stores the score of a WOD block of one of the user's workouts, or corrects it. Like
// sets, scores can be logged until the workout is deleted, so a finished one can still be
// fixed; they are not checked against the plan, which may have changed since.
func (s *Service) LogResult(ctx context.Context, userID, workoutID uuid.UUID, in ResultInput) (Result, error) {
	if err := checkResult(&in); err != nil {
		return Result{}, err
	}
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return Result{}, err
	}
	return s.workouts.LogResult(ctx, workoutID, in)
}

// DeleteResult removes a score from one of the user's workouts.
func (s *Service) DeleteResult(ctx context.Context, userID, workoutID, resultID uuid.UUID) error {
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	if err := s.workouts.DeleteResult(ctx, workoutID, resultID); errors.Is(err, ErrNotFound) {
		return errs.NotFound("result not found", err)
	} else if err != nil {
		return err
	}
	return nil
}

// checkResult validates a score against its format, in place: the values the format doesn't
// use are dropped, the ones it needs must be there and within bounds.
func checkResult(in *ResultInput) error {
	in.Notes = strings.TrimSpace(in.Notes)
	if in.BlockPosition < 1 {
		return errs.Invalid("block_position must be 1 or more")
	}
	if utf8.RuneCountInString(in.Notes) > 1000 {
		return errs.Invalid("the notes can have at most 1000 characters")
	}

	var msg string
	switch in.ScoreType {
	case scoreForTime:
		// Done within the cap: the time; capped: the reps done until then.
		in.Rounds, in.Calories = nil, nil
		switch {
		case in.TimeSeconds != nil && in.Reps != nil:
			return errs.Invalid("enter the time, or the reps if the time cap was hit, not both")
		case in.TimeSeconds != nil:
			msg = inRange("time", in.TimeSeconds, 1, maxResultTime)
		default:
			msg = required("reps", "enter the time, or the reps if the time cap was hit", in.Reps, maxResultCount)
		}
	case scoreAMRAP:
		in.TimeSeconds, in.Calories = nil, nil
		msg = required("rounds", "enter the rounds", in.Rounds, maxResultRounds)
		if msg == "" && in.Reps != nil {
			msg = inRange("reps", in.Reps, 0, maxResultCount)
		}
	case scoreMaxReps:
		in.TimeSeconds, in.Rounds, in.Calories = nil, nil, nil
		msg = required("reps", "enter the reps", in.Reps, maxResultCount)
	case scoreMaxCalories:
		in.TimeSeconds, in.Rounds, in.Reps = nil, nil, nil
		msg = required("calories", "enter the calories", in.Calories, maxResultCount)
	case scoreEMOM:
		in.TimeSeconds, in.Reps, in.Calories = nil, nil, nil
		msg = required("rounds", "enter the rounds completed", in.Rounds, maxResultRounds)
	default:
		return errs.Invalid("unknown score type")
	}
	if msg != "" {
		return errs.Invalid(msg)
	}
	return nil
}

// required checks a value the format needs: without it the message is missing, else it must
// be 0..max.
func required(what, missing string, v *int32, max int32) string {
	if v == nil {
		return missing
	}
	return inRange(what, v, 0, max)
}

// inRange checks that v lies within [min, max], naming it what in the message.
func inRange(what string, v *int32, min, max int32) string {
	if *v < min || *v > max {
		return fmt.Sprintf("the %s must be between %d and %d", what, min, max)
	}
	return ""
}
