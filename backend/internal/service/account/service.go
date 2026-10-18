// Package account signs users in.
package account

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service/account/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	GetUserByEmail(ctx context.Context, email string) (repos.GetUserByEmailRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (repos.GetUserByIDRow, error)
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Service implements the account usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// Account is the signed-in user as the frontend sees it.
type Account struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
