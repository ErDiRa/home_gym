// Package client manages clients: the rows of the users table, created and edited by admins.
package client

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListUsers(ctx context.Context) ([]repos.ListUsersRow, error)
	CreateUser(ctx context.Context, arg repos.CreateUserParams) (repos.CreateUserRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (repos.GetUserByIDRow, error)
	UpdateUser(ctx context.Context, arg repos.UpdateUserParams) (repos.UpdateUserRow, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (int64, error)
	ListClientTrainings(ctx context.Context, userID uuid.UUID) ([]repos.ListClientTrainingsRow, error)
	ListClientExerciseRatings(ctx context.Context, userID uuid.UUID) ([]repos.ListClientExerciseRatingsRow, error)
	AdminEmailExists(ctx context.Context, email string) (bool, error)
}

// Service implements the client usecases.
type Service struct {
	q        Queries
	hashCost int // bcrypt cost of new passwords
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service {
	return &Service{q: repos.New(pool), hashCost: bcrypt.DefaultCost}
}

// Input is a client for Create and Update.
type Input struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"` // only used (and checked) by Create
}

// Client is a client with their trainings; the row's fields sit at the top level of the JSON.
type Client struct {
	repos.GetUserByIDRow
	Trainings []repos.ListClientTrainingsRow `json:"trainings"`
}

// validate normalizes and validates the email and names of in. Create checks the password
// itself.
func validate(in *Input) error {
	// Login lowercases the email it looks up, so store it the same way.
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
	// ParseAddress also accepts forms like "Anna <anna@x.ch>"; requiring the bare address back
	// rejects those.
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email {
		return service.Invalid("a valid email is required")
	}
	if in.FirstName == "" || in.LastName == "" || tooLong(in.FirstName, 100) || tooLong(in.LastName, 100) {
		return service.Invalid("first_name and last_name are required (max 100 characters)")
	}
	return nil
}

// emailInUse is the message for an email that belongs to another client or to an admin.
const emailInUse = "this email is already in use"

// checkNotAdminEmail rejects emails of admins: login checks users before admins, so a
// client with an admin's email would lock that admin out.
func (s *Service) checkNotAdminEmail(ctx context.Context, email string) error {
	taken, err := s.q.AdminEmailExists(ctx, email)
	if err != nil {
		return err
	}
	if taken {
		return service.Conflict(emailInUse, nil)
	}
	return nil
}

// mapError turns a failed query into the error the caller sees.
func mapError(err error) error {
	switch {
	case isUniqueViolation(err):
		return service.Conflict(emailInUse, err)
	case errors.Is(err, pgx.ErrNoRows):
		return service.NotFound("client not found", err)
	default:
		return err
	}
}

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// isUniqueViolation reports whether err is a Postgres unique_violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
