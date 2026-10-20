// Package training manages the trainings users plan for themselves.
//
// A training belongs to one user and runs over whole weeks from a Monday. It has its own
// sessions (e.g. "Session 1"), which hold blocks, and blocks hold one or more exercises, plus a
// schedule: the sessions planned for each week, up to 7, repeats allowed. A block has a kind
// (single, superset, complex, wod, see block_kinds) that decides how many exercises it takes,
// whether sets, reps and load are set once on the block or per exercise, and whether the block
// is done for one score (a WOD: for time, AMRAP...). The same shape is sent to and returned
// from the API.
//
// The user's library holds session templates of the same shape; adding one to a training copies
// it (the client sends the copy), so the two are independent afterwards. Every usecase acts on
// the given user's data only; another user's looks missing.
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
	ListTrainingSessions(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingSessionsRow, error)
	ListTrainingBlocks(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingBlocksRow, error)
	ListTrainingExercises(ctx context.Context, trainingID uuid.UUID) ([]repos.ListTrainingExercisesRow, error)
	ListWeekSessions(ctx context.Context, trainingID uuid.UUID) ([]repos.ListWeekSessionsRow, error)
	CreateTraining(ctx context.Context, arg repos.CreateTrainingParams) (repos.CreateTrainingRow, error)
	UpdateTraining(ctx context.Context, arg repos.UpdateTrainingParams) (repos.UpdateTrainingRow, error)
	DeleteTraining(ctx context.Context, arg repos.DeleteTrainingParams) (int64, error)
	InsertSession(ctx context.Context, arg repos.InsertSessionParams) (uuid.UUID, error)
	UpdateTrainingSession(ctx context.Context, arg repos.UpdateTrainingSessionParams) (int64, error)
	DeleteTrainingSessionsExcept(ctx context.Context, arg repos.DeleteTrainingSessionsExceptParams) error
	DeleteTrainingBlocks(ctx context.Context, trainingID uuid.UUID) error
	InsertWeekSession(ctx context.Context, arg repos.InsertWeekSessionParams) error
	UpdateWeekSession(ctx context.Context, arg repos.UpdateWeekSessionParams) (int64, error)
	DeleteWeekSessionsExcept(ctx context.Context, arg repos.DeleteWeekSessionsExceptParams) error
	ListLibrarySessions(ctx context.Context, userID uuid.UUID) ([]repos.ListLibrarySessionsRow, error)
	GetLibrarySession(ctx context.Context, arg repos.GetLibrarySessionParams) (repos.GetLibrarySessionRow, error)
	ListSessionBlocks(ctx context.Context, sessionID uuid.UUID) ([]repos.ListSessionBlocksRow, error)
	ListSessionExercises(ctx context.Context, sessionID uuid.UUID) ([]repos.ListSessionExercisesRow, error)
	UpdateLibrarySession(ctx context.Context, arg repos.UpdateLibrarySessionParams) (int64, error)
	DeleteSessionBlocks(ctx context.Context, sessionID uuid.UUID) error
	DeleteLibrarySession(ctx context.Context, arg repos.DeleteLibrarySessionParams) (int64, error)
	InsertSessionBlock(ctx context.Context, arg repos.InsertSessionBlockParams) (uuid.UUID, error)
	InsertSessionExercise(ctx context.Context, arg repos.InsertSessionExerciseParams) (int64, error)
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

// Score is how a scored block (a WOD) is scored: the format (see the Score* constants) and its
// time in seconds, a time cap or the duration depending on the format, plus the interval of an
// EMOM. All nil on other blocks.
type Score struct {
	ScoreType       *string `json:"score_type"`
	TimeSeconds     *int32  `json:"time_seconds"`
	IntervalSeconds *int32  `json:"interval_seconds"`
}

// The formats of a scored block, as in the session_blocks.score_type CHECK.
const (
	ScoreForTime     = "for_time"     // as fast as possible; time = optional cap
	ScoreAMRAP       = "amrap"        // as many rounds and reps as possible; time = duration
	ScoreMaxReps     = "max_reps"     // as many reps as possible; time = optional duration
	ScoreMaxCalories = "max_calories" // as many calories as possible; time = optional duration
	ScoreEMOM        = "emom"         // every interval on the interval; time = duration
)

type BlockInput struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Sets *int32 `json:"sets"` // nil unless the kind shares sets; the rounds of a WOD (optional)
	Amount
	Score
	Notes     string          `json:"notes"`
	Exercises []ExerciseInput `json:"exercises"`
}

// SessionInput is a session of a training, or a library template.
type SessionInput struct {
	// ID is set for a training's sessions that already exist, so they keep their id; new ones
	// have none. Ignored for templates (the id is in the URL).
	ID     *uuid.UUID   `json:"id"`
	Name   string       `json:"name"`
	Notes  string       `json:"notes"`
	Blocks []BlockInput `json:"blocks"`
}

// SlotInput is one planned session of a week. Session is the index of the session in
// Input.Sessions, as new sessions have no id yet.
type SlotInput struct {
	// ID is set for slots that already exist, so they keep their id (workouts point at them).
	ID      *uuid.UUID `json:"id"`
	Session int        `json:"session"`
}

// maxWeeks and maxSlots bound a training: at most a year, at most 7 sessions a week.
const (
	maxWeeks = 52
	maxSlots = 7
)

// Input is a whole training for Create and Update, the tree at once.
type Input struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Active      bool           `json:"active"`
	StartDate   string         `json:"start_date"` // YYYY-MM-DD, a Monday
	Weeks       int32          `json:"weeks"`
	Sessions    []SessionInput `json:"sessions"`
	Schedule    [][]SlotInput  `json:"schedule"` // one list per week, len == Weeks

	start pgtype.Date // StartDate, parsed by validate
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
	Score
	Notes     string     `json:"notes"`
	Exercises []Exercise `json:"exercises"`
}

type Session struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Notes  string    `json:"notes"`
	Blocks []Block   `json:"blocks"`
}

// Slot is one planned session of a week.
type Slot struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
}

// Training is the whole training tree; the embedded row's fields sit at the top level.
type Training struct {
	repos.GetTrainingRow
	Sessions []Session `json:"sessions"`
	Schedule [][]Slot  `json:"schedule"` // one list per week
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
