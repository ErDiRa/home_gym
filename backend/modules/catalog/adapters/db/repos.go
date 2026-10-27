// Package db is the catalog module's Postgres adapter: the repositories the app layer saves
// through and the read model the HTTP layer lists from, over the queries sqlc generates into
// dbmodels.
package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/catalog/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/catalog/app"
)

// Repo implements app.ExerciseRepository, app.RatingRepository and app.LookupRepository.
type Repo struct {
	q *dbmodels.Queries
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Repo{q: dbmodels.New(pool)}
}

func (r *Repo) CreateExercise(ctx context.Context, userID uuid.UUID, in app.ExerciseInput) (app.Exercise, error) {
	e, err := r.q.CreateExercise(ctx, dbmodels.CreateExerciseParams{
		UserID: userID, Name: in.Name, Description: in.Description, Type: in.Type, VideoUrls: in.VideoURLs,
		JointIds: in.JointIDs, MuscleGroupIds: in.MuscleGroupIDs, FocusIds: in.FocusIDs,
	})
	if err != nil {
		return app.Exercise{}, mapError(err)
	}
	return app.Exercise{
		ID: e.ID, Name: e.Name, Description: e.Description, Type: e.Type, VideoURLs: e.VideoUrls,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}, nil
}

func (r *Repo) UpdateExercise(ctx context.Context, userID, id uuid.UUID, in app.ExerciseInput) (app.Exercise, error) {
	e, err := r.q.UpdateExercise(ctx, dbmodels.UpdateExerciseParams{
		ID: id, UserID: userID, Name: in.Name, Description: in.Description, Type: in.Type, VideoUrls: in.VideoURLs,
		JointIds: in.JointIDs, MuscleGroupIds: in.MuscleGroupIDs, FocusIds: in.FocusIDs,
	})
	if err != nil {
		return app.Exercise{}, mapError(err)
	}
	return app.Exercise{
		ID: e.ID, Name: e.Name, Description: e.Description, Type: e.Type, VideoURLs: e.VideoUrls,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}, nil
}

func (r *Repo) RateExercise(ctx context.Context, userID, exerciseID uuid.UUID, in app.RatingInput) (app.Rating, error) {
	f, err := r.q.UpsertExerciseFeedback(ctx, dbmodels.UpsertExerciseFeedbackParams{
		UserID: userID, ExerciseID: exerciseID, Rating: in.Rating, Notes: in.Notes,
	})
	if err != nil {
		return app.Rating{}, mapError(err)
	}
	return app.Rating{
		ID: f.ID, UserID: f.UserID, ExerciseID: f.ExerciseID, Rating: f.Rating, Notes: f.Notes, UpdatedAt: f.UpdatedAt,
	}, nil
}

func (r *Repo) CreateFocus(ctx context.Context, in app.FocusInput) (app.Focus, error) {
	f, err := r.q.CreateFocus(ctx, dbmodels.CreateFocusParams{Name: in.Name, Alias: in.Alias})
	if err != nil {
		return app.Focus{}, mapError(err)
	}
	return app.Focus{ID: f.ID, Name: f.Name, Alias: f.Alias}, nil
}

func (r *Repo) CreateRepUnit(ctx context.Context, in app.UnitInput) (app.Unit, error) {
	u, err := r.q.CreateRepUnit(ctx, dbmodels.CreateRepUnitParams{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue})
	if err != nil {
		return app.Unit{}, mapError(err)
	}
	return app.Unit{ID: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue}, nil
}

func (r *Repo) CreateLoadUnit(ctx context.Context, in app.UnitInput) (app.Unit, error) {
	u, err := r.q.CreateLoadUnit(ctx, dbmodels.CreateLoadUnitParams{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue})
	if err != nil {
		return app.Unit{}, mapError(err)
	}
	return app.Unit{
		ID: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue,
		KgPerUnit: u.KgPerUnit, PercentOfMax: u.PercentOfMax,
	}, nil
}

// mapError turns what Postgres reports into the app's errors: no row (the user's live exercise
// wasn't there), a foreign key violation (an unknown linked id) and a unique violation (a taken
// name).
func mapError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errors.Join(app.ErrNotFound, err)
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation:
		return errors.Join(app.ErrUnknownReference, err)
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
		return errors.Join(app.ErrDuplicate, err)
	default:
		return err
	}
}
