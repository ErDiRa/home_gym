package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// Candidates lists the user's WHOOP activities that started in [from, to) (the client's local day
// of the workout), newest first, for picking the one to link to the workout.
func (s *Service) Candidates(ctx context.Context, userID, workoutID uuid.UUID, from, to time.Time) ([]Activity, error) {
	if from.IsZero() || to.IsZero() || !from.Before(to) || to.Sub(from) > 48*time.Hour {
		return nil, errs.Invalid("from and to are required (RFC 3339), from before to, at most two days apart")
	}
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return nil, err
	}
	token, err := s.accessToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	activities, err := s.api.Activities(ctx, token, from, to)
	return activities, apiError(err)
}

// Link fetches a WHOOP activity and the recovery of the day before it started and stores them with
// one of the user's workouts. Linking again replaces them, e.g. once WHOOP has scored the
// activity.
func (s *Service) Link(ctx context.Context, userID, workoutID, whoopID uuid.UUID) (Data, error) {
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return Data{}, err
	}
	token, err := s.accessToken(ctx, userID)
	if err != nil {
		return Data{}, err
	}
	a, err := s.api.Activity(ctx, token, whoopID)
	if err != nil {
		return Data{}, apiError(err)
	}
	// The recovery is scored on waking, so the newest one in the day before the activity.
	recovery, err := s.api.LatestRecovery(ctx, token, a.Start.Add(-24*time.Hour), a.Start)
	if err != nil {
		return Data{}, apiError(err)
	}
	return s.data.SaveData(ctx, Data{WorkoutID: workoutID, Activity: a, Recovery: recovery})
}

// Data returns the WHOOP data linked to one of the user's workouts; nil when there is none (also
// for another user's workout).
func (s *Service) Data(ctx context.Context, userID, workoutID uuid.UUID) (*Data, error) {
	owns, err := s.tracking.UserOwnsWorkout(ctx, userID, workoutID)
	if err != nil || !owns {
		return nil, err
	}
	return s.data.Data(ctx, workoutID)
}

// Unlink removes the WHOOP data from one of the user's workouts.
func (s *Service) Unlink(ctx context.Context, userID, workoutID uuid.UUID) error {
	owns, err := s.tracking.UserOwnsWorkout(ctx, userID, workoutID)
	if err != nil {
		return err
	}
	deleted := false
	if owns {
		if deleted, err = s.data.DeleteData(ctx, workoutID); err != nil {
			return err
		}
	}
	if !deleted {
		return errs.NotFound("no WHOOP data linked", nil)
	}
	return nil
}

// checkWorkout makes sure the workout is one of the user's (the tracking module's).
func (s *Service) checkWorkout(ctx context.Context, userID, workoutID uuid.UUID) error {
	owns, err := s.tracking.UserOwnsWorkout(ctx, userID, workoutID)
	if err != nil {
		return err
	}
	if !owns {
		return errs.NotFound("workout not found", nil)
	}
	return nil
}
