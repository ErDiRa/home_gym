// Package db is the identity module's Postgres adapter: the users repository over the queries
// sqlc generates into dbmodels.
package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/identity/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/identity/app"
)

// UsersRepo implements app.UserRepository.
type UsersRepo struct {
	q *dbmodels.Queries
}

func NewUsersRepo(pool *pgxpool.Pool) *UsersRepo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &UsersRepo{q: dbmodels.New(pool)}
}

func (r *UsersRepo) UserByEmail(ctx context.Context, email string) (app.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.User{}, app.ErrUserNotFound
	}
	if err != nil {
		return app.User{}, err
	}
	return app.User{Account: app.Account{ID: u.ID, Email: u.Email}, PasswordHash: u.PasswordHash}, nil
}

func (r *UsersRepo) AccountByID(ctx context.Context, id uuid.UUID) (app.Account, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Account{}, app.ErrUserNotFound
	}
	if err != nil {
		return app.Account{}, err
	}
	return app.Account{ID: u.ID, Email: u.Email}, nil
}

// CreateUser adds a user with a bcrypt password hash. There is no sign-up: this is for tests and
// tools; users are otherwise created by hand.
func (r *UsersRepo) CreateUser(ctx context.Context, email, firstName, lastName, passwordHash string) (uuid.UUID, error) {
	return r.q.CreateUser(ctx, dbmodels.CreateUserParams{
		Email: email, FirstName: firstName, LastName: lastName, PasswordHash: passwordHash,
	})
}
