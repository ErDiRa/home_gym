package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/db"
	"github.com/erdira/homegym/backend/modules/planning/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/planning/app"
)

// Repo implements app.TrainingRepository, app.LibraryRepository and app.CycleRepository. Every
// write runs in one transaction (common/db.UpdateInTx).
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Repo{pool: pool}
}

// inTx runs fn with the queries in a transaction; the errors it returns are mapped to the app's.
func (r *Repo) inTx(ctx context.Context, fn func(q *dbmodels.Queries) error) error {
	return mapError(db.UpdateInTx(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		return fn(dbmodels.New(tx))
	}))
}

func (r *Repo) CreateTraining(ctx context.Context, userID uuid.UUID, in app.Input) (app.TrainingHeader, error) {
	var t app.TrainingHeader
	err := r.inTx(ctx, func(q *dbmodels.Queries) error {
		var err error
		t, err = createTraining(ctx, q, userID, in.CycleID, in)
		return err
	})
	return t, err
}

// createTraining saves a new training with its tree in cycleID, which must be one of the user's
// live cycles.
func createTraining(ctx context.Context, q *dbmodels.Queries, userID uuid.UUID, cycleID *uuid.UUID, in app.Input) (app.TrainingHeader, error) {
	if err := checkCycle(ctx, q, userID, cycleID); err != nil {
		return app.TrainingHeader{}, err
	}
	row, err := q.CreateTraining(ctx, dbmodels.CreateTrainingParams{
		UserID: userID, CycleID: cycleID, Name: in.Name, Description: in.Description, Active: in.Active,
		StartDate: in.Start(), Weeks: in.Weeks,
	})
	if err != nil {
		return app.TrainingHeader{}, err
	}
	ids, err := insertSessions(ctx, q, userID, row.ID, in.Sessions)
	if err != nil {
		return app.TrainingHeader{}, err
	}
	return app.TrainingHeader(row), writeSchedule(ctx, q, row.ID, in.Schedule, ids)
}

// UpdateTraining loads the training and replaces it with what update returns, in one transaction.
// The transaction may run (and call update) again after a conflict with a concurrent change.
func (r *Repo) UpdateTraining(ctx context.Context, userID, id uuid.UUID, update func(context.Context, app.Training) (app.Input, error)) (app.TrainingHeader, error) {
	var t app.TrainingHeader
	err := r.inTx(ctx, func(q *dbmodels.Queries) error {
		current, err := loadTraining(ctx, q, userID, id)
		if err != nil {
			return err
		}
		in, err := update(ctx, current)
		if err != nil {
			return err
		}
		if err := checkCycle(ctx, q, userID, in.CycleID); err != nil {
			return err
		}
		row, err := q.UpdateTraining(ctx, dbmodels.UpdateTrainingParams{
			ID: id, UserID: userID, CycleID: in.CycleID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.Start(), Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		t = app.TrainingHeader(row)
		ids, err := updateSessions(ctx, q, userID, id, in.Sessions)
		if err != nil {
			return err
		}
		return writeSchedule(ctx, q, id, in.Schedule, ids)
	})
	return t, err
}

func (r *Repo) Training(ctx context.Context, userID, id uuid.UUID) (app.Training, error) {
	return loadTraining(ctx, dbmodels.New(r.pool), userID, id)
}

func (r *Repo) DeleteTraining(ctx context.Context, userID, id uuid.UUID) error {
	n, err := dbmodels.New(r.pool).DeleteTraining(ctx, dbmodels.DeleteTrainingParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return app.ErrNotFound
	}
	return err
}

// checkCycle fails with app.ErrUnknownCycle unless cycleID is nil or one of the user's live cycles.
func checkCycle(ctx context.Context, q *dbmodels.Queries, userID uuid.UUID, cycleID *uuid.UUID) error {
	if cycleID == nil {
		return nil
	}
	ok, err := q.UserOwnsCycle(ctx, dbmodels.UserOwnsCycleParams{ID: *cycleID, UserID: userID})
	if err != nil {
		return err
	}
	if !ok {
		return app.ErrUnknownCycle
	}
	return nil
}

func (r *Repo) CreateSession(ctx context.Context, userID uuid.UUID, in app.SessionInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.inTx(ctx, func(q *dbmodels.Queries) error {
		var err error
		if id, err = q.InsertSession(ctx, dbmodels.InsertSessionParams{UserID: userID, Name: in.Name, Notes: in.Notes}); err != nil {
			return err
		}
		return insertBlocks(ctx, q, id, in.Blocks)
	})
	return id, err
}

func (r *Repo) UpdateSession(ctx context.Context, userID, id uuid.UUID, in app.SessionInput) error {
	return r.inTx(ctx, func(q *dbmodels.Queries) error {
		n, err := q.UpdateLibrarySession(ctx, dbmodels.UpdateLibrarySessionParams{ID: id, UserID: userID, Name: in.Name, Notes: in.Notes})
		if err != nil {
			return err
		}
		if n == 0 {
			return app.ErrNotFound
		}
		if err := q.DeleteSessionBlocks(ctx, id); err != nil {
			return err
		}
		return insertBlocks(ctx, q, id, in.Blocks)
	})
}

func (r *Repo) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	n, err := dbmodels.New(r.pool).DeleteLibrarySession(ctx, dbmodels.DeleteLibrarySessionParams{ID: id, UserID: userID})
	if err == nil && n == 0 {
		return app.ErrNotFound
	}
	return err
}

func (r *Repo) CreateCycle(ctx context.Context, userID uuid.UUID, in app.CycleInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.inTx(ctx, func(q *dbmodels.Queries) error {
		row, err := q.CreateCycle(ctx, dbmodels.CreateCycleParams{
			UserID: userID, Name: in.Name, Description: in.Description, StartDate: in.Start(), Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		id = row.ID
		return writeGoals(ctx, q, id, in.Goals)
	})
	return id, err
}

func (r *Repo) UpdateCycle(ctx context.Context, userID, id uuid.UUID, in app.CycleInput) error {
	return r.inTx(ctx, func(q *dbmodels.Queries) error {
		if _, err := q.UpdateCycle(ctx, dbmodels.UpdateCycleParams{
			ID: id, UserID: userID, Name: in.Name, Description: in.Description, StartDate: in.Start(), Weeks: in.Weeks,
		}); err != nil {
			return err
		}
		return writeGoals(ctx, q, id, in.Goals)
	})
}

func (r *Repo) DeleteCycle(ctx context.Context, userID, id uuid.UUID, withTrainings bool) error {
	return r.inTx(ctx, func(q *dbmodels.Queries) error {
		n, err := q.DeleteCycle(ctx, dbmodels.DeleteCycleParams{ID: id, UserID: userID})
		if err != nil {
			return err
		}
		if n == 0 {
			return app.ErrNotFound
		}
		// A soft delete doesn't fire the foreign key's ON DELETE SET NULL.
		if withTrainings {
			return q.DeleteCycleTrainings(ctx, id)
		}
		return q.ClearCycleFromTrainings(ctx, id)
	})
}

// ImportCycle saves the cycle, its goals and its trainings (in the new cycle) in one transaction.
func (r *Repo) ImportCycle(ctx context.Context, userID uuid.UUID, cycle app.CycleInput, trainings []app.Input) (app.ImportedCycle, error) {
	var imported app.ImportedCycle
	err := r.inTx(ctx, func(q *dbmodels.Queries) error {
		row, err := q.CreateCycle(ctx, dbmodels.CreateCycleParams{
			UserID: userID, Name: cycle.Name, Description: cycle.Description, StartDate: cycle.Start(), Weeks: cycle.Weeks,
		})
		if err != nil {
			return err
		}
		if err := writeGoals(ctx, q, row.ID, cycle.Goals); err != nil {
			return err
		}
		imported = app.ImportedCycle{CycleHeader: app.CycleHeader(row), Trainings: make([]app.TrainingHeader, 0, len(trainings))}
		for _, t := range trainings {
			training, err := createTraining(ctx, q, userID, &row.ID, t)
			if err != nil {
				return err
			}
			imported.Trainings = append(imported.Trainings, training)
		}
		return nil
	})
	return imported, err
}

// writeGoals replaces a cycle's goals with the given ones, in order.
func writeGoals(ctx context.Context, q *dbmodels.Queries, cycleID uuid.UUID, goals []app.GoalInput) error {
	if err := q.DeleteCycleGoals(ctx, cycleID); err != nil {
		return err
	}
	for i, g := range goals {
		if err := q.InsertCycleGoal(ctx, dbmodels.InsertCycleGoalParams{
			CycleID: cycleID, Position: int32(i + 1), Title: g.Title, Target: g.Target, Status: g.Status, Notes: g.Notes,
		}); err != nil {
			return err
		}
	}
	return nil
}
