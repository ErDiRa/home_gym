// Package cycle manages the cycles users plan for themselves.
//
// A cycle is a planned period of whole weeks from a Monday with goals, e.g. "Mobility, 8 weeks:
// better hips, better elbow extension". It groups trainings (trainings.cycle_id), each in at most
// one cycle; its period is its own, so a training may lie partly or wholly outside it (the cycle
// can be moved later) and several of its trainings may run in the same week. Putting a training
// into a cycle, and importing a cycle with its trainings, are usecases of the training service.
// Every usecase acts on the given user's data only; another user's looks missing.
package cycle

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListCycles(ctx context.Context, userID uuid.UUID) ([]repos.ListCyclesRow, error)
	GetCycle(ctx context.Context, arg repos.GetCycleParams) (repos.GetCycleRow, error)
	ListCycleGoals(ctx context.Context, cycleID uuid.UUID) ([]repos.ListCycleGoalsRow, error)
	ListCycleTrainings(ctx context.Context, cycleID uuid.UUID) ([]repos.ListCycleTrainingsRow, error)
	CreateCycle(ctx context.Context, arg repos.CreateCycleParams) (repos.CreateCycleRow, error)
	UpdateCycle(ctx context.Context, arg repos.UpdateCycleParams) (repos.UpdateCycleRow, error)
	DeleteCycle(ctx context.Context, arg repos.DeleteCycleParams) (int64, error)
	DeleteCycleGoals(ctx context.Context, cycleID uuid.UUID) error
	InsertCycleGoal(ctx context.Context, arg repos.InsertCycleGoalParams) error
	ClearCycleFromTrainings(ctx context.Context, cycleID uuid.UUID) error
	DeleteCycleTrainings(ctx context.Context, cycleID uuid.UUID) error
}

// TxFunc runs fn in a transaction, committing if it returns nil and rolling back otherwise.
type TxFunc func(ctx context.Context, fn func(q Queries) error) error

// Service implements the cycle usecases.
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

// The statuses of a goal, as in the cycle_goals.status CHECK.
const (
	GoalOpen     = "open"
	GoalAchieved = "achieved"
	GoalMissed   = "missed"
)

// maxWeeks and maxGoals bound a cycle: at most a year, at most 20 goals.
const (
	maxWeeks = 52
	maxGoals = 20
)

// GoalInput is one goal of a cycle. Goals are rewritten on every save, in the order sent.
type GoalInput struct {
	ID     *uuid.UUID `json:"id"`     // ignored on save; the client sends back what it loaded
	Title  string     `json:"title"`  // required, e.g. "Better hips"
	Target string     `json:"target"` // optional, something to check at the end
	Status string     `json:"status"` // open (default when empty), achieved or missed
	Notes  string     `json:"notes"`
}

// Input is a whole cycle for Create and Update; the training service validates the cycle of a
// cycle import with it too (Validate).
type Input struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	StartDate   string      `json:"start_date"` // YYYY-MM-DD, a Monday
	Weeks       int32       `json:"weeks"`
	Goals       []GoalInput `json:"goals"`

	Start pgtype.Date `json:"-"` // StartDate, parsed by Validate
}

// Cycle is a cycle with its goals and trainings, as the dashboard loads it; the embedded row's
// fields sit at the top level.
type Cycle struct {
	repos.GetCycleRow
	Goals     []repos.ListCycleGoalsRow     `json:"goals"`
	Trainings []repos.ListCycleTrainingsRow `json:"trainings"`
}

// load reads one of the user's cycles with its goals and trainings.
func load(ctx context.Context, q Queries, userID, id uuid.UUID) (Cycle, error) {
	row, err := q.GetCycle(ctx, repos.GetCycleParams{ID: id, UserID: userID})
	if err != nil {
		return Cycle{}, err
	}
	return withGoalsAndTrainings(ctx, q, row)
}

// withGoalsAndTrainings completes a cycle's row with its goals and trainings.
func withGoalsAndTrainings(ctx context.Context, q Queries, row repos.GetCycleRow) (Cycle, error) {
	goals, err := q.ListCycleGoals(ctx, row.ID)
	if err != nil {
		return Cycle{}, err
	}
	trainings, err := q.ListCycleTrainings(ctx, row.ID)
	if err != nil {
		return Cycle{}, err
	}
	return Cycle{GetCycleRow: row, Goals: orEmpty(goals), Trainings: orEmpty(trainings)}, nil
}

// writeGoals replaces a cycle's goals with the given ones, in order.
func writeGoals(ctx context.Context, q Queries, cycleID uuid.UUID, goals []GoalInput) error {
	if err := q.DeleteCycleGoals(ctx, cycleID); err != nil {
		return err
	}
	for i, g := range goals {
		err := q.InsertCycleGoal(ctx, repos.InsertCycleGoalParams{
			CycleID: cycleID, Position: int32(i + 1), Title: g.Title, Target: g.Target, Status: g.Status, Notes: g.Notes,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// mapError turns a failed query into the error the caller sees: a missing row is the cycle.
// Errors that are already a *service.Error pass through.
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return service.NotFound("cycle not found", err)
	}
	return err
}

// orEmpty turns a nil slice into an empty one, so it encodes as [] rather than null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }
