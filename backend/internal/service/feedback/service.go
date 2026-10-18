// Package feedback is what a user reports about their exercises: their ratings, and stats over
// the workouts they did.
package feedback

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	UpsertExerciseFeedback(ctx context.Context, arg repos.UpsertExerciseFeedbackParams) (repos.ExerciseFeedback, error)
	ListExerciseRatings(ctx context.Context, userID uuid.UUID) ([]repos.ListExerciseRatingsRow, error)
	GetExerciseStats(ctx context.Context, userID uuid.UUID) ([]repos.GetExerciseStatsRow, error)
}

// Service implements the feedback usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }
