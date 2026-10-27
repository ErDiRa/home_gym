// Package db is the tracking module's Postgres adapter: the repositories the app layer saves
// through and the read model the HTTP layer lists from, over the queries sqlc generates into
// dbmodels.
package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
	"github.com/erdira/homegym/backend/modules/tracking/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/tracking/app"
)

// Repo implements app.WorkoutRepository and app.MaxRepository.
type Repo struct {
	q *dbmodels.Queries
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Repo{q: dbmodels.New(pool)}
}

func (r *Repo) CreateWorkout(ctx context.Context, userID uuid.UUID, slot planning.Slot) (uuid.UUID, error) {
	id, err := r.q.CreateWorkout(ctx, dbmodels.CreateWorkoutParams{
		UserID: userID, TrainingID: &slot.TrainingID, WeekSessionID: &slot.ID, SessionID: &slot.SessionID,
		SessionName: slot.SessionName,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		return uuid.Nil, errors.Join(app.ErrSlotInProgress, err)
	}
	return id, err
}

func (r *Repo) SlotWorkoutInProgress(ctx context.Context, userID, slotID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.GetSlotWorkoutInProgress(ctx, dbmodels.GetSlotWorkoutInProgressParams{WeekSessionID: &slotID, UserID: userID})
	return id, notFound(err)
}

func (r *Repo) FinishWorkout(ctx context.Context, userID, id uuid.UUID, notes string, completedAt *time.Time) error {
	n, err := r.q.FinishWorkout(ctx, dbmodels.FinishWorkoutParams{ID: id, UserID: userID, Notes: notes, CompletedAt: completedAt})
	return rowsAffected(n, err)
}

func (r *Repo) DeleteWorkout(ctx context.Context, userID, id uuid.UUID) error {
	return rowsAffected(r.q.DeleteWorkout(ctx, dbmodels.DeleteWorkoutParams{ID: id, UserID: userID}))
}

func (r *Repo) FinishedWorkouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]app.WorkoutSummary, error) {
	rows, err := r.q.ListWorkouts(ctx, dbmodels.ListWorkoutsParams{UserID: userID, FromTime: &from, ToTime: &to})
	if err != nil {
		return nil, err
	}
	out := make([]app.WorkoutSummary, len(rows))
	for i, w := range rows {
		out[i] = workoutSummary(w)
	}
	return out, nil
}

// workoutSummary maps a finished workout (completed_at is set).
func workoutSummary(w dbmodels.ListWorkoutsRow) app.WorkoutSummary {
	return app.WorkoutSummary{
		ID: w.ID, TrainingID: w.TrainingID, WeekSessionID: w.WeekSessionID, SessionID: w.SessionID, SessionName: w.SessionName,
		StartedAt: w.StartedAt, CompletedAt: *w.CompletedAt, Notes: w.Notes, SetCount: w.SetCount,
	}
}

func (r *Repo) UserOwnsWorkout(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	return r.q.UserOwnsWorkout(ctx, dbmodels.UserOwnsWorkoutParams{ID: id, UserID: userID})
}

func (r *Repo) LogSet(ctx context.Context, workoutID uuid.UUID, in app.SetInput) (app.LoggedSet, error) {
	s, err := r.q.UpsertWorkoutSet(ctx, dbmodels.UpsertWorkoutSetParams{
		WorkoutID: workoutID, ExerciseID: in.ExerciseID,
		BlockPosition: in.BlockPosition, ExercisePosition: in.ExercisePosition, SetNumber: in.SetNumber,
		RepsValue: in.RepsValue, RepsUnit: *in.RepsUnit, PerSide: in.PerSide, LoadValue: in.LoadValue, LoadUnit: in.LoadUnit,
	})
	return app.LoggedSet(s), err
}

func (r *Repo) DeleteSet(ctx context.Context, workoutID, setID uuid.UUID) error {
	return rowsAffected(r.q.DeleteWorkoutSet(ctx, dbmodels.DeleteWorkoutSetParams{ID: setID, WorkoutID: workoutID}))
}

func (r *Repo) LogResult(ctx context.Context, workoutID uuid.UUID, in app.ResultInput) (app.Result, error) {
	res, err := r.q.UpsertWorkoutResult(ctx, dbmodels.UpsertWorkoutResultParams{
		WorkoutID: workoutID, BlockPosition: in.BlockPosition, ScoreType: in.ScoreType,
		TimeSeconds: in.TimeSeconds, Rounds: in.Rounds, Reps: in.Reps, Calories: in.Calories, Rx: in.Rx, Notes: in.Notes,
	})
	return app.Result(res), err
}

func (r *Repo) DeleteResult(ctx context.Context, workoutID, resultID uuid.UUID) error {
	return rowsAffected(r.q.DeleteWorkoutResult(ctx, dbmodels.DeleteWorkoutResultParams{ID: resultID, WorkoutID: workoutID}))
}

func (r *Repo) CreateMax(ctx context.Context, userID, exerciseID uuid.UUID, in app.MaxInput, achievedOn time.Time) (uuid.UUID, error) {
	return r.q.CreateExerciseMax(ctx, dbmodels.CreateExerciseMaxParams{
		UserID: userID, ExerciseID: exerciseID, LoadValue: in.LoadValue, LoadUnit: in.LoadUnit, AchievedOn: achievedOn,
		WorkoutSetID: in.WorkoutSetID, Notes: in.Notes,
	})
}

func (r *Repo) UserOwnsSet(ctx context.Context, userID, setID, exerciseID uuid.UUID) (bool, error) {
	return r.q.UserOwnsSet(ctx, dbmodels.UserOwnsSetParams{UserID: userID, ID: setID, ExerciseID: exerciseID})
}

func (r *Repo) DeleteMax(ctx context.Context, userID, id uuid.UUID) error {
	return rowsAffected(r.q.DeleteExerciseMax(ctx, dbmodels.DeleteExerciseMaxParams{ID: id, UserID: userID}))
}

// rowsAffected turns "no row changed" into app.ErrNotFound.
func rowsAffected(n int64, err error) error {
	if err == nil && n == 0 {
		return app.ErrNotFound
	}
	return err
}

// notFound turns "no row" into app.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.Join(app.ErrNotFound, err)
	}
	return err
}
