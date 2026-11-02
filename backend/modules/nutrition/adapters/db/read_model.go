package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/modules/nutrition/adapters/db/dbmodels"
	nutritionhttp "github.com/erdira/homegym/backend/modules/nutrition/api/http"
)

// ReadModel lists the users' bodies, mapped straight to the HTTP responses: reading is plain data
// retrieval, no business logic.
type ReadModel struct {
	q *dbmodels.Queries
}

var _ nutritionhttp.ReadModel = (*ReadModel)(nil)

func NewReadModel(pool *pgxpool.Pool) *ReadModel {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &ReadModel{q: dbmodels.New(pool)}
}

// Profile returns the user's profile, or nil.
func (m *ReadModel) Profile(ctx context.Context, userID uuid.UUID) (*nutritionhttp.Profile, error) {
	row, err := m.q.GetProfile(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &nutritionhttp.Profile{
		HeightCm: row.HeightCm, BirthDate: openapi_types.Date{Time: row.BirthDate}, Sex: nutritionhttp.Sex(row.Sex),
		ActivityFactor: row.ActivityFactor, UpdatedAt: row.UpdatedAt,
	}, nil
}

// Weights lists the user's weights, newest first.
func (m *ReadModel) Weights(ctx context.Context, userID uuid.UUID) ([]nutritionhttp.BodyWeight, error) {
	rows, err := m.q.ListBodyWeights(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]nutritionhttp.BodyWeight, len(rows))
	for i, r := range rows {
		out[i] = nutritionhttp.BodyWeight{
			Id: r.ID, MeasuredOn: openapi_types.Date{Time: r.MeasuredOn}, WeightKg: r.WeightKg, BodyFatPercent: r.BodyFatPercent, Notes: r.Notes,
		}
	}
	return out, nil
}
