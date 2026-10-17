// Package account signs users and admins in. They are separate accounts in separate
// tables; an Account covers both.
package account

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service/account/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	GetUserByEmail(ctx context.Context, email string) (repos.GetUserByEmailRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (repos.GetUserByIDRow, error)
	GetAdminByEmail(ctx context.Context, email string) (repos.GetAdminByEmailRow, error)
	GetAdminByID(ctx context.Context, id uuid.UUID) (repos.GetAdminByIDRow, error)
	IsAdmin(ctx context.Context, id uuid.UUID) (bool, error)
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Service implements the account usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// Account is the signed-in account as the frontend sees it, for users and admins alike.
type Account struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	IsAdmin bool      `json:"is_admin"`
}

// Role is the account's role for the token: auth.RoleAdmin or auth.RoleUser.
func (a Account) Role() string {
	if a.IsAdmin {
		return auth.RoleAdmin
	}
	return auth.RoleUser
}
