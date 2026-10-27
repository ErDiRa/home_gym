// Package db is the planning module's Postgres adapter: the repositories the app layer saves
// through and the read model the HTTP layer lists from, over the queries sqlc generates into
// dbmodels.
package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/modules/planning/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/planning/app"
)

// A training's tree (sessions → blocks → exercises) and its week plan, read and written with the
// same queries for trainings and library templates.

// loadTraining reads one of the user's trainings with its sessions, blocks and exercises in order,
// and its schedule with one list per week.
func loadTraining(ctx context.Context, q *dbmodels.Queries, userID, id uuid.UUID) (app.Training, error) {
	row, err := q.GetTraining(ctx, dbmodels.GetTrainingParams{ID: id, UserID: userID})
	if err != nil {
		return app.Training{}, mapError(err)
	}
	sessions, err := q.ListTrainingSessions(ctx, id)
	if err != nil {
		return app.Training{}, err
	}
	blocks, err := q.ListTrainingBlocks(ctx, id)
	if err != nil {
		return app.Training{}, err
	}
	exercises, err := q.ListTrainingExercises(ctx, id)
	if err != nil {
		return app.Training{}, err
	}
	slots, err := q.ListWeekSessions(ctx, id)
	if err != nil {
		return app.Training{}, err
	}

	t := app.Training{TrainingHeader: app.TrainingHeader(row), Sessions: make([]app.Session, 0, len(sessions))}
	blocksBySession := groupBlocks(blocks, exercises)
	for _, s := range sessions {
		t.Sessions = append(t.Sessions, app.Session{ID: s.ID, Name: s.Name, Notes: s.Notes, Blocks: orEmpty(blocksBySession[s.ID])})
	}
	t.Schedule = make([][]app.Slot, row.Weeks)
	for i := range t.Schedule {
		t.Schedule[i] = []app.Slot{}
	}
	for _, s := range slots {
		// Slots of weeks past the end (only possible by hand in the database) are left out.
		if s.Week <= row.Weeks {
			t.Schedule[s.Week-1] = append(t.Schedule[s.Week-1], app.Slot{ID: s.ID, SessionID: s.SessionID})
		}
	}
	return t, nil
}

// loadTemplateBlocks reads a library template's blocks with their exercises.
func loadTemplateBlocks(ctx context.Context, q *dbmodels.Queries, sessionID uuid.UUID) ([]app.Block, error) {
	blocks, err := q.ListSessionBlocks(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	exercises, err := q.ListSessionExercises(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// The rows have the same fields as a training's, so they share groupBlocks.
	tb := make([]dbmodels.ListTrainingBlocksRow, len(blocks))
	for i, b := range blocks {
		tb[i] = dbmodels.ListTrainingBlocksRow(b)
	}
	te := make([]dbmodels.ListTrainingExercisesRow, len(exercises))
	for i, e := range exercises {
		te[i] = dbmodels.ListTrainingExercisesRow(e)
	}
	return orEmpty(groupBlocks(tb, te)[sessionID]), nil
}

// groupBlocks builds the blocks with their exercises, grouped by session. The rows come back
// ordered by position within their parent, so appending keeps the order.
func groupBlocks(blocks []dbmodels.ListTrainingBlocksRow, exercises []dbmodels.ListTrainingExercisesRow) map[uuid.UUID][]app.Block {
	exercisesByBlock := map[uuid.UUID][]app.Exercise{}
	for _, e := range exercises {
		exercisesByBlock[e.BlockID] = append(exercisesByBlock[e.BlockID], app.Exercise{
			ExerciseID: e.ExerciseID, Sets: e.Sets, Tempo: e.Tempo, Notes: e.Notes,
			Amount: app.Amount{
				RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
				LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit,
			},
		})
	}
	blocksBySession := map[uuid.UUID][]app.Block{}
	for _, b := range blocks {
		blocksBySession[b.SessionID] = append(blocksBySession[b.SessionID], app.Block{
			ID: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets, Notes: b.Notes,
			Score: app.Score{ScoreType: b.ScoreType, TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds},
			Amount: app.Amount{
				RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
				LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit,
			},
			Exercises: orEmpty(exercisesByBlock[b.ID]),
		})
	}
	return blocksBySession
}

// insertSessions writes the sessions, blocks and exercises of a new training of the user, in input
// order, and returns their ids by index. Session ids in the input are ignored: a new training has
// no sessions yet.
func insertSessions(ctx context.Context, q *dbmodels.Queries, userID, trainingID uuid.UUID, sessions []app.SessionInput) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(sessions))
	for i, s := range sessions {
		id, err := q.InsertSession(ctx, dbmodels.InsertSessionParams{
			UserID: userID, TrainingID: &trainingID, Position: int32(i + 1), Name: s.Name, Notes: s.Notes,
		})
		if err != nil {
			return nil, err
		}
		if err := insertBlocks(ctx, q, id, s.Blocks); err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

// errForeignID is returned when a session or slot id of an update doesn't belong to the training;
// the app layer checks that first, so this only guards the database.
var errForeignID = errors.New("session or slot of another training")

// updateSessions replaces the sessions of an existing training and returns their ids by index:
// sessions missing from the input are deleted (with their slots), sessions with an id are updated
// in place (keeping their id), sessions without one are inserted. All blocks and exercises are
// deleted and re-inserted.
func updateSessions(ctx context.Context, q *dbmodels.Queries, userID, trainingID uuid.UUID, sessions []app.SessionInput) ([]uuid.UUID, error) {
	keep := []uuid.UUID{}
	for _, s := range sessions {
		if s.ID != nil {
			keep = append(keep, *s.ID)
		}
	}
	if err := q.DeleteTrainingSessionsExcept(ctx, dbmodels.DeleteTrainingSessionsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
		return nil, err
	}
	// Blocks cascade to their exercises; both are re-inserted below in input order.
	if err := q.DeleteTrainingBlocks(ctx, trainingID); err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(sessions))
	for i, s := range sessions {
		position := int32(i + 1)
		if s.ID != nil {
			n, err := q.UpdateTrainingSession(ctx, dbmodels.UpdateTrainingSessionParams{
				ID: *s.ID, TrainingID: trainingID, Position: position, Name: s.Name, Notes: s.Notes,
			})
			if err != nil {
				return nil, err
			}
			if n == 0 {
				return nil, errForeignID
			}
			ids[i] = *s.ID
		} else {
			id, err := q.InsertSession(ctx, dbmodels.InsertSessionParams{
				UserID: userID, TrainingID: &trainingID, Position: position, Name: s.Name, Notes: s.Notes,
			})
			if err != nil {
				return nil, err
			}
			ids[i] = id
		}
		if err := insertBlocks(ctx, q, ids[i], s.Blocks); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// writeSchedule replaces a training's week plan: slots missing from the input are deleted, slots
// with an id are updated in place (keeping their id), slots without one are inserted. sessionIDs
// maps the slots' session indexes to the saved sessions.
func writeSchedule(ctx context.Context, q *dbmodels.Queries, trainingID uuid.UUID, schedule [][]app.SlotInput, sessionIDs []uuid.UUID) error {
	keep := []uuid.UUID{}
	for _, week := range schedule {
		for _, slot := range week {
			if slot.ID != nil {
				keep = append(keep, *slot.ID)
			}
		}
	}
	if err := q.DeleteWeekSessionsExcept(ctx, dbmodels.DeleteWeekSessionsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
		return err
	}
	// (training, week, position) is unique, deferred to the commit, so slots may swap places.
	for w, week := range schedule {
		for p, slot := range week {
			number, position, sessionID := int32(w+1), int32(p+1), sessionIDs[slot.Session]
			if slot.ID == nil {
				if err := q.InsertWeekSession(ctx, dbmodels.InsertWeekSessionParams{
					TrainingID: trainingID, Week: number, Position: position, SessionID: sessionID,
				}); err != nil {
					return err
				}
				continue
			}
			n, err := q.UpdateWeekSession(ctx, dbmodels.UpdateWeekSessionParams{
				ID: *slot.ID, TrainingID: trainingID, Week: number, Position: position, SessionID: sessionID,
			})
			if err != nil {
				return err
			}
			if n == 0 {
				return errForeignID
			}
		}
	}
	return nil
}

// insertBlocks inserts a session's blocks and their exercises, in input order.
func insertBlocks(ctx context.Context, q *dbmodels.Queries, sessionID uuid.UUID, blocks []app.BlockInput) error {
	for j, b := range blocks {
		blockID, err := q.InsertSessionBlock(ctx, dbmodels.InsertSessionBlockParams{
			SessionID: sessionID, Position: int32(j + 1), Kind: b.Kind, Name: b.Name, Sets: b.Sets,
			RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
			LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit,
			ScoreType: b.ScoreType, TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds, Notes: b.Notes,
		})
		if err != nil {
			return err
		}
		for k, e := range b.Exercises {
			if err := q.InsertSessionExercise(ctx, dbmodels.InsertSessionExerciseParams{
				BlockID: blockID, ExerciseID: e.ExerciseID, Position: int32(k + 1), Sets: e.Sets,
				RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
				LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, Tempo: e.Tempo, Notes: e.Notes,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// mapError turns what Postgres reports into the app's errors: no row (the user's live training,
// template or cycle wasn't there) and a foreign key violation (a referenced row is gone).
func mapError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errors.Join(app.ErrNotFound, err)
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation:
		return errors.Join(app.ErrUnknownReference, err)
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
