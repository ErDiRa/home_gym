// Package maxes keeps a user's 1RM history per exercise. The newest entry is the current max,
// from which the tracker turns loads planned in % 1RM into a weight.
package maxes

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListExerciseMaxes(ctx context.Context, userID uuid.UUID) ([]repos.ListExerciseMaxesRow, error)
	GetLoadUnit(ctx context.Context, id uuid.UUID) (repos.GetLoadUnitRow, error)
	UserOwnsSet(ctx context.Context, arg repos.UserOwnsSetParams) (bool, error)
	CreateExerciseMax(ctx context.Context, arg repos.CreateExerciseMaxParams) (repos.CreateExerciseMaxRow, error)
	DeleteExerciseMax(ctx context.Context, arg repos.DeleteExerciseMaxParams) (int64, error)
}

// Service implements the 1RM usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }
