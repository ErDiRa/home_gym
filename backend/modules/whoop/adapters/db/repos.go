// Package db is the whoop module's Postgres adapter: the users' WHOOP connections and the WHOOP
// data linked to workouts, over the queries sqlc generates into dbmodels.
package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/whoop/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/whoop/app"
)

// Repo implements app.ConnectionRepository and app.DataRepository.
type Repo struct {
	q *dbmodels.Queries
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Repo{q: dbmodels.New(pool)}
}

func (r *Repo) Connection(ctx context.Context, userID uuid.UUID) (app.Tokens, error) {
	c, err := r.q.GetConnection(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Tokens{}, app.ErrNotConnected
	}
	if err != nil {
		return app.Tokens{}, err
	}
	return app.Tokens{AccessToken: c.AccessToken, RefreshToken: c.RefreshToken, ExpiresAt: c.ExpiresAt}, nil
}

func (r *Repo) SaveConnection(ctx context.Context, userID uuid.UUID, t app.Tokens) error {
	return r.q.SaveConnection(ctx, dbmodels.SaveConnectionParams{
		UserID: userID, AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, ExpiresAt: t.ExpiresAt,
	})
}

func (r *Repo) DeleteConnection(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteConnection(ctx, userID)
}

func (r *Repo) SaveData(ctx context.Context, d app.Data) (app.Data, error) {
	arg := dbmodels.SaveWorkoutDataParams{
		WorkoutID: d.WorkoutID, WhoopWorkoutID: d.ID, SportName: d.SportName, StartAt: d.Start, EndAt: d.End,
		ScoreState: d.ScoreState, Strain: d.Strain, AvgHr: d.AvgHR, MaxHr: d.MaxHR, Kilojoule: d.Kilojoule, ZoneMs: d.ZoneMs,
	}
	if rec := d.Recovery; rec != nil {
		arg.RecoveryScore, arg.RestingHr, arg.HrvMs = &rec.Score, &rec.RestingHR, &rec.HrvMs
	}
	row, err := r.q.SaveWorkoutData(ctx, arg)
	if err != nil {
		return app.Data{}, err
	}
	return dataFromRow(row), nil
}

func (r *Repo) Data(ctx context.Context, workoutID uuid.UUID) (*app.Data, error) {
	row, err := r.q.GetWorkoutData(ctx, workoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := dataFromRow(row)
	return &d, nil
}

func (r *Repo) DeleteData(ctx context.Context, workoutID uuid.UUID) (bool, error) {
	n, err := r.q.DeleteWorkoutData(ctx, workoutID)
	return n > 0, err
}

// dataFromRow maps a stored row; the recovery is set when its score is.
func dataFromRow(row dbmodels.WhoopWorkoutWhoopDatum) app.Data {
	d := app.Data{
		WorkoutID: row.WorkoutID,
		Activity: app.Activity{
			ID: row.WhoopWorkoutID, SportName: row.SportName, Start: row.StartAt, End: row.EndAt, ScoreState: row.ScoreState,
			Strain: row.Strain, AvgHR: row.AvgHr, MaxHR: row.MaxHr, Kilojoule: row.Kilojoule, ZoneMs: row.ZoneMs,
		},
		FetchedAt: row.FetchedAt,
	}
	if row.RecoveryScore != nil {
		d.Recovery = &app.Recovery{Score: *row.RecoveryScore}
		if row.RestingHr != nil {
			d.Recovery.RestingHR = *row.RestingHr
		}
		if row.HrvMs != nil {
			d.Recovery.HrvMs = *row.HrvMs
		}
	}
	return d
}
