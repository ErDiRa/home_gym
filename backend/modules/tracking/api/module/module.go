// Package module implements the tracking module's contract (client.Tracking).
package module

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/tracking/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

type Tracking struct {
	q *dbmodels.Queries
}

var _ client.Tracking = (*Tracking)(nil)

func New(pool *pgxpool.Pool) *Tracking {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Tracking{q: dbmodels.New(pool)}
}

func (t *Tracking) UserOwnsWorkout(ctx context.Context, userID, workoutID uuid.UUID) (bool, error) {
	return t.q.UserOwnsWorkout(ctx, dbmodels.UserOwnsWorkoutParams{ID: workoutID, UserID: userID})
}
