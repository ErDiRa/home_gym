// Package module implements the catalog module's contract (client.Catalog). Its reads are plain
// data retrieval, like the read model's.
package module

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/catalog/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/catalog/api/module/client"
)

type Catalog struct {
	q *dbmodels.Queries
}

var _ client.Catalog = (*Catalog)(nil)

func New(pool *pgxpool.Pool) *Catalog {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Catalog{q: dbmodels.New(pool)}
}

func (c *Catalog) Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]client.Exercise, error) {
	out := make(map[uuid.UUID]client.Exercise, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := c.q.ListExercisesByIDs(ctx, dbmodels.ListExercisesByIDsParams{UserID: userID, Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, e := range rows {
		out[e.ID] = client.Exercise{ID: e.ID, Name: e.Name, Type: e.Type, Deleted: e.Deleted}
	}
	return out, nil
}

func (c *Catalog) BlockKinds(ctx context.Context) ([]client.BlockKind, error) {
	rows, err := c.q.ListBlockKinds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]client.BlockKind, len(rows))
	for i, k := range rows {
		out[i] = client.BlockKind{
			Code: k.Code, Name: k.Name, MinExercises: k.MinExercises, MaxExercises: k.MaxExercises,
			SharedSets: k.SharedSets, SharedReps: k.SharedReps, SharedLoad: k.SharedLoad, Scored: k.Scored,
		}
	}
	return out, nil
}

func (c *Catalog) Units(ctx context.Context) (client.Units, error) {
	rep, err := c.q.ListAllRepUnits(ctx)
	if err != nil {
		return client.Units{}, err
	}
	load, err := c.q.ListAllLoadUnits(ctx)
	if err != nil {
		return client.Units{}, err
	}
	units := client.Units{Rep: make([]client.Unit, len(rep)), Load: make([]client.Unit, len(load))}
	for i, u := range rep {
		units.Rep[i] = client.Unit{ID: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue, Deleted: u.Deleted}
	}
	for i, u := range load {
		units.Load[i] = client.Unit{
			ID: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue,
			KgPerUnit: u.KgPerUnit, PercentOfMax: u.PercentOfMax, Deleted: u.Deleted,
		}
	}
	return units, nil
}

func (c *Catalog) ExerciseProfiles(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]client.ExerciseProfile, error) {
	out := make(map[uuid.UUID]client.ExerciseProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := c.q.ListExerciseProfiles(ctx, dbmodels.ListExerciseProfilesParams{UserID: userID, Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, e := range rows {
		out[e.ID] = client.ExerciseProfile(e)
	}
	return out, nil
}
