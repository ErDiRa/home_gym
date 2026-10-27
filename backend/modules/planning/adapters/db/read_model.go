package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	"github.com/erdira/homegym/backend/modules/planning/adapters/db/dbmodels"
	planninghttp "github.com/erdira/homegym/backend/modules/planning/api/http"
	"github.com/erdira/homegym/backend/modules/planning/app"
)

// Catalog is what the read model reads of the catalog module (its contract): the names of the
// exercises and units trainings plan with.
type Catalog interface {
	Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error)
	Units(ctx context.Context) (catalog.Units, error)
}

// ReadModel lists what the dashboard and the forms show, mapped straight to the HTTP responses:
// reading is plain data retrieval, no business logic. The names of exercises and units are the
// catalog's, read through its contract.
type ReadModel struct {
	q       *dbmodels.Queries
	catalog Catalog
}

var _ planninghttp.ReadModel = (*ReadModel)(nil)

func NewReadModel(pool *pgxpool.Pool, catalog Catalog) *ReadModel {
	if pool == nil {
		panic("pool cannot be nil")
	}
	if catalog == nil {
		panic("catalog cannot be nil")
	}
	return &ReadModel{q: dbmodels.New(pool), catalog: catalog}
}

// Trainings lists the user's trainings with their size and cycle, the active ones first.
func (m *ReadModel) Trainings(ctx context.Context, userID uuid.UUID) ([]planninghttp.Training, error) {
	rows, err := m.q.ListTrainings(ctx, userID)
	return mapRows(rows, err, func(t dbmodels.ListTrainingsRow) planninghttp.Training {
		return planninghttp.Training{
			Id: t.ID, CycleId: t.CycleID, CycleName: t.CycleName, Name: t.Name, Description: t.Description, Active: t.Active,
			StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks, UpdatedAt: t.UpdatedAt,
			SessionCount: t.SessionCount, PlannedCount: t.PlannedCount,
		}
	})
}

// Training is one of the user's trainings with its sessions, blocks, exercises and schedule.
func (m *ReadModel) Training(ctx context.Context, userID, id uuid.UUID) (planninghttp.TrainingDetail, error) {
	t, err := loadTraining(ctx, m.q, userID, id)
	if errors.Is(err, app.ErrNotFound) {
		return planninghttp.TrainingDetail{}, errs.NotFound("training not found", err)
	}
	if err != nil {
		return planninghttp.TrainingDetail{}, err
	}
	var blocks []app.Block
	for _, s := range t.Sessions {
		blocks = append(blocks, s.Blocks...)
	}
	names, err := m.names(ctx, userID, blocks)
	if err != nil {
		return planninghttp.TrainingDetail{}, err
	}
	return planninghttp.TrainingDetail{
		Id: t.ID, CycleId: t.CycleID, Name: t.Name, Description: t.Description, Active: t.Active,
		StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		Sessions: mapSlice(t.Sessions, func(s app.Session) planninghttp.TrainingSession {
			return planninghttp.TrainingSession{Id: s.ID, Name: s.Name, Notes: s.Notes, Blocks: names.blocks(s.Blocks)}
		}),
		Schedule: mapSlice(t.Schedule, func(week []app.Slot) []planninghttp.Slot {
			return mapSlice(week, func(s app.Slot) planninghttp.Slot { return planninghttp.Slot{Id: s.ID, SessionId: s.SessionID} })
		}),
	}, nil
}

// LibrarySessions lists the user's session templates with their size, by name.
func (m *ReadModel) LibrarySessions(ctx context.Context, userID uuid.UUID) ([]planninghttp.LibrarySession, error) {
	rows, err := m.q.ListLibrarySessions(ctx, userID)
	return mapRows(rows, err, func(s dbmodels.ListLibrarySessionsRow) planninghttp.LibrarySession {
		return planninghttp.LibrarySession{
			Id: s.ID, Name: s.Name, Notes: s.Notes, UpdatedAt: s.UpdatedAt, BlockCount: s.BlockCount, ExerciseCount: s.ExerciseCount,
		}
	})
}

// LibrarySession is one of the user's live session templates with its blocks and exercises.
func (m *ReadModel) LibrarySession(ctx context.Context, userID, id uuid.UUID) (planninghttp.LibrarySessionDetail, error) {
	row, err := m.q.GetLibrarySession(ctx, dbmodels.GetLibrarySessionParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return planninghttp.LibrarySessionDetail{}, errs.NotFound("session not found", err)
	}
	if err != nil {
		return planninghttp.LibrarySessionDetail{}, err
	}
	blocks, err := loadTemplateBlocks(ctx, m.q, id)
	if err != nil {
		return planninghttp.LibrarySessionDetail{}, err
	}
	names, err := m.names(ctx, userID, blocks)
	if err != nil {
		return planninghttp.LibrarySessionDetail{}, err
	}
	return planninghttp.LibrarySessionDetail{
		Id: row.ID, Name: row.Name, Notes: row.Notes, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Blocks: names.blocks(blocks),
	}, nil
}

// Cycles lists the user's live cycles with their goal and training counts, the latest first.
func (m *ReadModel) Cycles(ctx context.Context, userID uuid.UUID) ([]planninghttp.CycleSummary, error) {
	rows, err := m.q.ListCycles(ctx, userID)
	return mapRows(rows, err, func(c dbmodels.ListCyclesRow) planninghttp.CycleSummary {
		return planninghttp.CycleSummary{
			Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
			Weeks: c.Weeks, UpdatedAt: c.UpdatedAt, GoalCount: c.GoalCount, TrainingCount: c.TrainingCount,
		}
	})
}

// Cycle is one of the user's live cycles with its goals and trainings.
func (m *ReadModel) Cycle(ctx context.Context, userID, id uuid.UUID) (planninghttp.Cycle, error) {
	c, err := m.q.GetCycle(ctx, dbmodels.GetCycleParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return planninghttp.Cycle{}, errs.NotFound("cycle not found", err)
	}
	if err != nil {
		return planninghttp.Cycle{}, err
	}
	goals, err := m.q.ListCycleGoals(ctx, id)
	if err != nil {
		return planninghttp.Cycle{}, err
	}
	trainings, err := m.q.ListCycleTrainings(ctx, id)
	if err != nil {
		return planninghttp.Cycle{}, err
	}
	return planninghttp.Cycle{
		Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
		Weeks: c.Weeks, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		Goals: mapSlice(goals, func(g dbmodels.ListCycleGoalsRow) planninghttp.CycleGoal {
			return planninghttp.CycleGoal{
				Id: g.ID, Position: g.Position, Title: g.Title, Target: g.Target, Status: planninghttp.GoalStatus(g.Status), Notes: g.Notes,
			}
		}),
		Trainings: mapSlice(trainings, func(t dbmodels.ListCycleTrainingsRow) planninghttp.CycleTraining {
			return planninghttp.CycleTraining{
				Id: t.ID, Name: t.Name, Active: t.Active, StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks,
			}
		}),
	}, nil
}

// catalogNames holds the names of the exercises and units some blocks plan with.
type catalogNames struct {
	exercises map[uuid.UUID]catalog.Exercise
	repUnits  map[uuid.UUID]string
	loadUnits map[uuid.UUID]string
}

// names reads the names of the blocks' exercises and units from the catalog.
func (m *ReadModel) names(ctx context.Context, userID uuid.UUID, blocks []app.Block) (catalogNames, error) {
	var ids []uuid.UUID
	for _, b := range blocks {
		for _, e := range b.Exercises {
			ids = append(ids, e.ExerciseID)
		}
	}
	exercises, err := m.catalog.Exercises(ctx, userID, ids)
	if err != nil {
		return catalogNames{}, err
	}
	units, err := m.catalog.Units(ctx)
	if err != nil {
		return catalogNames{}, err
	}
	n := catalogNames{exercises: exercises, repUnits: map[uuid.UUID]string{}, loadUnits: map[uuid.UUID]string{}}
	for _, u := range units.Rep {
		n.repUnits[u.ID] = u.Name
	}
	for _, u := range units.Load {
		n.loadUnits[u.ID] = u.Name
	}
	return n, nil
}

// unitName is the name of a unit, or "" without one.
func unitName(names map[uuid.UUID]string, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return names[*id]
}

func (n catalogNames) blocks(blocks []app.Block) []planninghttp.TrainingBlock {
	return mapSlice(blocks, func(b app.Block) planninghttp.TrainingBlock {
		return planninghttp.TrainingBlock{
			Id: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets,
			RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, RepsUnitName: unitName(n.repUnits, b.RepsUnit), PerSide: b.PerSide,
			LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, LoadUnitName: unitName(n.loadUnits, b.LoadUnit), LoadBodyweight: b.LoadBodyweight,
			ScoreType: (*planninghttp.ScoreType)(b.ScoreType), TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds,
			Notes: b.Notes,
			Exercises: mapSlice(b.Exercises, func(e app.Exercise) planninghttp.TrainingExercise {
				ex := n.exercises[e.ExerciseID]
				return planninghttp.TrainingExercise{
					ExerciseId: e.ExerciseID, Name: ex.Name, Type: planninghttp.ExerciseType(ex.Type), Sets: e.Sets,
					RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, RepsUnitName: unitName(n.repUnits, e.RepsUnit), PerSide: e.PerSide,
					LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, LoadUnitName: unitName(n.loadUnits, e.LoadUnit), LoadBodyweight: e.LoadBodyweight,
					Tempo: e.Tempo, Notes: e.Notes,
				}
			}),
		}
	})
}

// date is a database date as an API date.
func date(t time.Time) openapi_types.Date { return openapi_types.Date{Time: t} }

// mapSlice maps every element with f; the result is never nil, so it encodes as [].
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// mapRows maps the rows of a query with f, or passes its error on.
func mapRows[T, U any](rows []T, err error, f func(T) U) ([]U, error) {
	if err != nil {
		return nil, err
	}
	return mapSlice(rows, f), nil
}
