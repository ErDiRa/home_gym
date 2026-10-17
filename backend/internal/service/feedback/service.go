// Package feedback is what clients report about exercises: their ratings, and stats over
// the workouts they logged.
package feedback

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service/feedback/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	UpsertExerciseFeedback(ctx context.Context, arg repos.UpsertExerciseFeedbackParams) (repos.ExerciseFeedback, error)
	GetExerciseStats(ctx context.Context, userID uuid.UUID) ([]repos.GetExerciseStatsRow, error)
}

// Service implements the feedback usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// isForeignKeyViolation reports whether err is a Postgres foreign_key_violation.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
