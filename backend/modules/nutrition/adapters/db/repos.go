// Package db is the nutrition module's Postgres adapter: the users' bodies and macros, over the
// queries sqlc generates into dbmodels.
package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/nutrition/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/nutrition/app"
)

// Repo implements app.Repository.
type Repo struct {
	q *dbmodels.Queries
}

var _ app.Repository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Repo{q: dbmodels.New(pool)}
}

func (r *Repo) Profile(ctx context.Context, userID uuid.UUID) (*app.Profile, error) {
	row, err := r.q.GetProfile(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &app.Profile{
		HeightCm: row.HeightCm, BirthDate: row.BirthDate, Sex: app.Sex(row.Sex), ActivityFactor: row.ActivityFactor,
	}, nil
}

func (r *Repo) SaveProfile(ctx context.Context, userID uuid.UUID, p app.Profile) error {
	_, err := r.q.SaveProfile(ctx, dbmodels.SaveProfileParams{
		UserID: userID, HeightCm: p.HeightCm, BirthDate: p.BirthDate, Sex: string(p.Sex), ActivityFactor: p.ActivityFactor,
	})
	return err
}

func (r *Repo) Weights(ctx context.Context, userID uuid.UUID) ([]app.Weight, error) {
	rows, err := r.q.ListBodyWeights(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]app.Weight, len(rows))
	for i, row := range rows {
		out[i] = weightFromRow(row)
	}
	return out, nil
}

func (r *Repo) SaveWeight(ctx context.Context, userID uuid.UUID, w app.Weight) (app.Weight, error) {
	row, err := r.q.SaveBodyWeight(ctx, dbmodels.SaveBodyWeightParams{
		UserID: userID, MeasuredOn: w.MeasuredOn, WeightKg: w.WeightKg, BodyFatPercent: w.BodyFatPercent, Notes: w.Notes,
	})
	if err != nil {
		return app.Weight{}, err
	}
	return weightFromRow(row), nil
}

func (r *Repo) DeleteWeight(ctx context.Context, userID, id uuid.UUID) error {
	n, err := r.q.DeleteBodyWeight(ctx, dbmodels.DeleteBodyWeightParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return app.ErrNotFound
	}
	return err
}

func (r *Repo) Macros(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]app.Macros, error) {
	rows, err := r.q.ListDailyMacros(ctx, dbmodels.ListDailyMacrosParams{UserID: userID, FromDay: from, ToDay: to})
	if err != nil {
		return nil, err
	}
	out := make([]app.Macros, len(rows))
	for i, row := range rows {
		out[i] = macrosFromRow(row)
	}
	return out, nil
}

func (r *Repo) SaveMacros(ctx context.Context, userID uuid.UUID, m app.Macros) (app.Macros, error) {
	row, err := r.q.SaveDailyMacros(ctx, dbmodels.SaveDailyMacrosParams{
		UserID: userID, Day: m.Day, ProteinG: m.ProteinG, FatG: m.FatG, CarbsG: m.CarbsG, Notes: m.Notes,
	})
	if err != nil {
		return app.Macros{}, err
	}
	return macrosFromRow(row), nil
}

func (r *Repo) DeleteMacros(ctx context.Context, userID uuid.UUID, day time.Time) error {
	n, err := r.q.DeleteDailyMacros(ctx, dbmodels.DeleteDailyMacrosParams{UserID: userID, Day: day})
	if err == nil && n == 0 {
		return app.ErrNotFound
	}
	return err
}

func weightFromRow(row dbmodels.NutritionBodyWeight) app.Weight {
	return app.Weight{ID: row.ID, MeasuredOn: row.MeasuredOn, WeightKg: row.WeightKg, BodyFatPercent: row.BodyFatPercent, Notes: row.Notes}
}

func macrosFromRow(row dbmodels.NutritionDailyMacro) app.Macros {
	return app.Macros{
		Day: row.Day, ProteinG: row.ProteinG, FatG: row.FatG, CarbsG: row.CarbsG, Kcal: row.Kcal,
		Notes: row.Notes, Updated: row.UpdatedAt,
	}
}
