// Package training manages the trainings users plan for themselves.
//
// A training belongs to one user and is a tree: units (e.g. "Day A", done in turn, as many
// sessions per week as the training asks for) hold blocks, and blocks hold one or more
// exercises. A block has a kind (single, superset, complex, see block_kinds) that decides how
// many exercises it takes and whether sets, reps and load are set once on the block or per
// exercise. The same shape is sent to and returned from the API. Every usecase acts on the
// given user's trainings only; another user's look missing.
package training

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListTrainings(ctx context.Context, userID uuid.UUID) ([]repos.ListTrainingsRow, error)
	GetTraining(ctx context.Context, arg repos.GetTrainingParams) (repos.GetTrainingRow, error)
	GetActiveTrainingID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	ListTrainingUnits(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingUnitsRow, error)
	ListTrainingBlocks(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingBlocksRow, error)
	ListTrainingExercises(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingExercisesRow, error)
	CreateTraining(ctx context.Context, arg repos.CreateTrainingParams) (repos.CreateTrainingRow, error)
	UpdateTraining(ctx context.Context, arg repos.UpdateTrainingParams) (repos.UpdateTrainingRow, error)
	DeactivateUserTrainings(ctx context.Context, userID uuid.UUID) error
	DeleteTraining(ctx context.Context, arg repos.DeleteTrainingParams) (int64, error)
	InsertTrainingUnit(ctx context.Context, arg repos.InsertTrainingUnitParams) (uuid.UUID, error)
	UpdateTrainingUnit(ctx context.Context, arg repos.UpdateTrainingUnitParams) (int64, error)
	DeleteTrainingUnitsExcept(ctx context.Context, arg repos.DeleteTrainingUnitsExceptParams) error
	DeleteTrainingBlocks(ctx context.Context, trainingID uuid.UUID) error
	InsertTrainingBlock(ctx context.Context, arg repos.InsertTrainingBlockParams) (uuid.UUID, error)
	InsertTrainingExercise(ctx context.Context, arg repos.InsertTrainingExerciseParams) (int64, error)
	ListBlockKinds(ctx context.Context) ([]repos.ListBlockKindsRow, error)
	ListRepUnits(ctx context.Context) ([]repos.ListRepUnitsRow, error)
	ListLoadUnits(ctx context.Context) ([]repos.ListLoadUnitsRow, error)
}

// TxFunc runs fn in a transaction, committing if it returns nil and rolling back otherwise.
type TxFunc func(ctx context.Context, fn func(q Queries) error) error

// Service implements the training usecases.
type Service struct {
	q    Queries
	inTx TxFunc
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service {
	q := repos.New(pool)
	return &Service{
		q: q,
		inTx: func(ctx context.Context, fn func(q Queries) error) error {
			return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return fn(q.WithTx(tx)) })
		},
	}
}

// Amount is the planned reps and load of a block or exercise. A value is nil when not set;
// a max turns it into a range (e.g. 8-10 reps). Units are rep_units / load_units ids.
type Amount struct {
	RepsValue *float64   `json:"reps_value"`
	RepsMax   *float64   `json:"reps_max"`
	RepsUnit  *uuid.UUID `json:"reps_unit"`
	PerSide   bool       `json:"per_side"`
	LoadValue *float64   `json:"load_value"`
	LoadMax   *float64   `json:"load_max"`
	LoadUnit  *uuid.UUID `json:"load_unit"`
}

// clearReps and clearLoad empty the part of an amount that the block's kind keeps on the other
// side (block or exercises).
func (a *Amount) clearReps() { a.RepsValue, a.RepsMax, a.RepsUnit, a.PerSide = nil, nil, nil, false }
func (a *Amount) clearLoad() { a.LoadValue, a.LoadMax, a.LoadUnit = nil, nil, nil }

// ExerciseInput is one exercise in a block. The embedded Amount's fields appear at the top
// level of the JSON object.
type ExerciseInput struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Sets       *int32    `json:"sets"` // nil when the block's kind shares sets
	Amount
	Tempo string `json:"tempo"` // free text, e.g. '31X1'
	Notes string `json:"notes"`
}

type BlockInput struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Sets *int32 `json:"sets"` // nil unless the kind shares sets
	Amount
	Notes     string          `json:"notes"`
	Exercises []ExerciseInput `json:"exercises"`
}

type UnitInput struct {
	// ID is set for units that already exist, so they keep their id; new units have none.
	ID     *uuid.UUID   `json:"id"`
	Name   string       `json:"name"`
	Notes  string       `json:"notes"`
	Blocks []BlockInput `json:"blocks"`
}

// Input is a whole training for Create and Update, the tree at once.
type Input struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Active          bool        `json:"active"`
	StartDate       string      `json:"start_date"` // YYYY-MM-DD, first day of the period
	EndDate         string      `json:"end_date"`   // YYYY-MM-DD, last day of the period
	SessionsPerWeek int32       `json:"sessions_per_week"`
	Units           []UnitInput `json:"units"`

	start, end pgtype.Date // StartDate and EndDate, parsed by validate
}

// AmountWithUnits is an Amount with the names of its units, for display.
type AmountWithUnits struct {
	Amount
	RepsUnitName string `json:"reps_unit_name"`
	LoadUnitName string `json:"load_unit_name"`
}

type Exercise struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Sets       *int32    `json:"sets"`
	AmountWithUnits
	Tempo string `json:"tempo"`
	Notes string `json:"notes"`
}

type Block struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
	Name string    `json:"name"`
	Sets *int32    `json:"sets"`
	AmountWithUnits
	Notes     string     `json:"notes"`
	Exercises []Exercise `json:"exercises"`
}

type Unit struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Notes  string    `json:"notes"`
	Blocks []Block   `json:"blocks"`
}

// Training is the whole training tree; the embedded row's fields sit at the top level.
type Training struct {
	repos.GetTrainingRow
	Units []Unit `json:"units"`
}

// mapError turns a failed query into the error the caller sees. Errors that are already a
// *service.Error pass through.
func mapError(err error) error {
	var svcErr *service.Error
	switch {
	case err == nil, errors.As(err, &svcErr):
		return err
	case errors.Is(err, pgx.ErrNoRows):
		return service.NotFound("training not found", err)
	// Kinds and units are checked during validation, so a broken reference here is an exercise.
	case isForeignKeyViolation(err):
		return service.InvalidCause("unknown exercise", err)
	default:
		return err
	}
}

// orEmpty turns a nil slice into an empty one, so it encodes as [] rather than null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// isForeignKeyViolation reports whether err is a Postgres foreign_key_violation.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
