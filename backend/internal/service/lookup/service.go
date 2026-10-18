// Package lookup manages the lookup tables users pick from: joints, muscle groups and
// focuses for exercises, and block kinds and the units of reps and load for trainings.
// Trainings are validated against the rows of the latter, so a new kind or unit needs no
// code change.
package lookup

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListJoints(ctx context.Context) ([]repos.ListJointsRow, error)
	ListMuscleGroups(ctx context.Context) ([]repos.ListMuscleGroupsRow, error)
	ListFocuses(ctx context.Context) ([]repos.ListFocusesRow, error)
	CreateFocus(ctx context.Context, arg repos.CreateFocusParams) (repos.CreateFocusRow, error)
	ListBlockKinds(ctx context.Context) ([]repos.ListBlockKindsRow, error)
	ListRepUnits(ctx context.Context) ([]repos.ListRepUnitsRow, error)
	CreateRepUnit(ctx context.Context, arg repos.CreateRepUnitParams) (repos.CreateRepUnitRow, error)
	ListLoadUnits(ctx context.Context) ([]repos.ListLoadUnitsRow, error)
	CreateLoadUnit(ctx context.Context, arg repos.CreateLoadUnitParams) (repos.CreateLoadUnitRow, error)
}

// Service implements the lookup usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// isUniqueViolation reports whether err is a Postgres unique_violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
