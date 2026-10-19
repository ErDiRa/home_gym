// Package workout tracks the sessions a user does of their trainings.
//
// A training plans the sessions of each of its weeks instead of fixed days. A workout is one
// session done: the user starts it from a planned slot of the week whenever there is time, logs
// every set as they go and finishes it. Several trainings can be active side by side (e.g.
// strength, warm-ups and mobility); the week view shows one of them at a time: its planned
// sessions of the week next to the finished workouts, which mark their slots done.
package workout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListActiveTrainings(ctx context.Context, userID uuid.UUID) ([]repos.ListActiveTrainingsRow, error)
	ListWeekPlan(ctx context.Context, arg repos.ListWeekPlanParams) ([]repos.ListWeekPlanRow, error)
	GetWeekSession(ctx context.Context, arg repos.GetWeekSessionParams) (repos.GetWeekSessionRow, error)
	CreateWorkout(ctx context.Context, arg repos.CreateWorkoutParams) (repos.Workout, error)
	GetWorkout(ctx context.Context, arg repos.GetWorkoutParams) (repos.Workout, error)
	GetCurrentWorkout(ctx context.Context, userID uuid.UUID) (repos.Workout, error)
	FinishWorkout(ctx context.Context, arg repos.FinishWorkoutParams) (repos.Workout, error)
	DeleteWorkout(ctx context.Context, arg repos.DeleteWorkoutParams) (int64, error)
	ListWorkouts(ctx context.Context, arg repos.ListWorkoutsParams) ([]repos.ListWorkoutsRow, error)
	ListWorkoutSets(ctx context.Context, workoutID uuid.UUID) ([]repos.ListWorkoutSetsRow, error)
	UpsertWorkoutSet(ctx context.Context, arg repos.UpsertWorkoutSetParams) (repos.WorkoutSet, error)
	DeleteWorkoutSet(ctx context.Context, arg repos.DeleteWorkoutSetParams) (int64, error)
	ListRepUnits(ctx context.Context) ([]repos.ListRepUnitsRow, error)
	ListLoadUnits(ctx context.Context) ([]repos.ListLoadUnitsRow, error)
}

// Service implements the workout usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// Workout is a session done, with the sets logged so far, in the order of the session.
type Workout struct {
	ID            uuid.UUID                  `json:"id"`
	TrainingID    *uuid.UUID                 `json:"training_id"`     // nil once the training is deleted
	WeekSessionID *uuid.UUID                 `json:"week_session_id"` // the slot; nil once it is removed
	SessionID     *uuid.UUID                 `json:"session_id"`      // nil once the session is removed
	SessionName   string                     `json:"session_name"`
	StartedAt     pgtype.Timestamptz         `json:"started_at"`
	CompletedAt   pgtype.Timestamptz         `json:"completed_at"` // null while in progress
	Notes         string                     `json:"notes"`
	Sets          []repos.ListWorkoutSetsRow `json:"sets"`
}

// fromRow turns a workout row and its sets into a Workout.
func fromRow(w repos.Workout, sets []repos.ListWorkoutSetsRow) Workout {
	return Workout{
		ID: w.ID, TrainingID: w.TrainingID, WeekSessionID: w.WeekSessionID, SessionID: w.SessionID,
		SessionName: w.SessionName, StartedAt: w.StartedAt, CompletedAt: w.CompletedAt, Notes: w.Notes, Sets: sets,
	}
}

// withSets reads the sets of a workout row into a Workout.
func withSets(ctx context.Context, q Queries, w repos.Workout) (Workout, error) {
	sets, err := q.ListWorkoutSets(ctx, w.ID)
	if err != nil {
		return Workout{}, err
	}
	return fromRow(w, sets), nil
}

// mapError turns a failed query into the error the caller sees. Errors that are already a
// *service.Error pass through.
func mapError(err error) error {
	var svcErr *service.Error
	switch {
	case err == nil, errors.As(err, &svcErr):
		return err
	case errors.Is(err, pgx.ErrNoRows):
		return service.NotFound("workout not found", err)
	default:
		return err
	}
}

// period checks a [from, to) range of finished workouts to list.
func period(from, to time.Time) (pgtype.Timestamptz, pgtype.Timestamptz, error) {
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, service.Invalid("from and to are required (RFC 3339), from before to")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, service.Invalid("the period can be at most a year")
	}
	return pgtype.Timestamptz{Time: from, Valid: true}, pgtype.Timestamptz{Time: to, Valid: true}, nil
}

// isUniqueViolation reports whether err is a Postgres unique_violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
