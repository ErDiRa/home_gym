package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// insertSessions writes the sessions, blocks and exercises of a new training of the user, in
// input order, and returns their ids by index. Session ids in the input are ignored: a new
// training has no sessions yet.
func insertSessions(ctx context.Context, q Queries, userID, trainingID uuid.UUID, sessions []SessionInput) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(sessions))
	for i, s := range sessions {
		id, err := q.InsertSession(ctx, repos.InsertSessionParams{
			UserID: userID, TrainingID: &trainingID, Position: int32(i + 1), Name: s.Name, Notes: s.Notes,
		})
		if err != nil {
			return nil, err
		}
		if err := insertBlocks(ctx, q, userID, id, s.Blocks); err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

// updateSessions replaces the sessions of an existing training and returns their ids by index:
// sessions missing from the input are deleted (with their slots), sessions with an id are
// updated in place (keeping their id), sessions without one are inserted. All blocks and
// exercises are deleted and re-inserted.
func updateSessions(ctx context.Context, q Queries, userID, trainingID uuid.UUID, sessions []SessionInput) ([]uuid.UUID, error) {
	keep := []uuid.UUID{}
	for _, s := range sessions {
		if s.ID != nil {
			keep = append(keep, *s.ID)
		}
	}
	if err := q.DeleteTrainingSessionsExcept(ctx, repos.DeleteTrainingSessionsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
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
			n, err := q.UpdateTrainingSession(ctx, repos.UpdateTrainingSessionParams{
				ID: *s.ID, TrainingID: trainingID, Position: position, Name: s.Name, Notes: s.Notes,
			})
			if err != nil {
				return nil, err
			}
			// The id belongs to another training, or to none: refuse rather than move it.
			if n == 0 {
				return nil, service.Invalid("unknown session")
			}
			ids[i] = *s.ID
		} else {
			id, err := q.InsertSession(ctx, repos.InsertSessionParams{
				UserID: userID, TrainingID: &trainingID, Position: position, Name: s.Name, Notes: s.Notes,
			})
			if err != nil {
				return nil, err
			}
			ids[i] = id
		}
		if err := insertBlocks(ctx, q, userID, ids[i], s.Blocks); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// writeSchedule replaces a training's week plan: slots missing from the input are deleted,
// slots with an id are updated in place (keeping their id), slots without one are inserted.
// sessionIDs maps the slots' session indexes to the saved sessions.
func writeSchedule(ctx context.Context, q Queries, trainingID uuid.UUID, schedule [][]SlotInput, sessionIDs []uuid.UUID) error {
	keep := []uuid.UUID{}
	for _, week := range schedule {
		for _, slot := range week {
			if slot.ID != nil {
				keep = append(keep, *slot.ID)
			}
		}
	}
	if err := q.DeleteWeekSessionsExcept(ctx, repos.DeleteWeekSessionsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
		return err
	}
	// (training, week, position) is unique, deferred to the commit, so slots may swap places.
	for w, week := range schedule {
		for p, slot := range week {
			number, position, sessionID := int32(w+1), int32(p+1), sessionIDs[slot.Session]
			if slot.ID == nil {
				if err := q.InsertWeekSession(ctx, repos.InsertWeekSessionParams{
					TrainingID: trainingID, Week: number, Position: position, SessionID: sessionID,
				}); err != nil {
					return err
				}
				continue
			}
			n, err := q.UpdateWeekSession(ctx, repos.UpdateWeekSessionParams{
				ID: *slot.ID, TrainingID: trainingID, Week: number, Position: position, SessionID: sessionID,
			})
			if err != nil {
				return err
			}
			if n == 0 {
				return service.Invalid("unknown slot in the schedule")
			}
		}
	}
	return nil
}

// insertBlocks inserts a session's blocks and their exercises, in input order. The exercises
// must be the user's own.
func insertBlocks(ctx context.Context, q Queries, userID, sessionID uuid.UUID, blocks []BlockInput) error {
	for j, b := range blocks {
		blockID, err := q.InsertSessionBlock(ctx, repos.InsertSessionBlockParams{
			SessionID: sessionID, Position: int32(j + 1), Kind: b.Kind, Name: b.Name, Sets: b.Sets,
			RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
			LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, Notes: b.Notes,
		})
		if err != nil {
			return err
		}
		for k, e := range b.Exercises {
			n, err := q.InsertSessionExercise(ctx, repos.InsertSessionExerciseParams{
				BlockID: blockID, ExerciseID: e.ExerciseID, UserID: userID, Position: int32(k + 1), Sets: e.Sets,
				RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
				LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, Tempo: e.Tempo, Notes: e.Notes,
			})
			if err != nil {
				return err
			}
			// Another user's, a deleted or an unknown exercise.
			if n == 0 {
				return service.Invalid("unknown exercise")
			}
		}
	}
	return nil
}

// copyBlocks turns loaded blocks into new ones for insertBlocks.
func copyBlocks(blocks []Block) []BlockInput {
	out := make([]BlockInput, 0, len(blocks))
	for _, b := range blocks {
		block := BlockInput{Kind: b.Kind, Name: b.Name, Sets: b.Sets, Amount: b.Amount, Notes: b.Notes}
		for _, e := range b.Exercises {
			block.Exercises = append(block.Exercises, ExerciseInput{
				ExerciseID: e.ExerciseID, Sets: e.Sets, Amount: e.Amount, Tempo: e.Tempo, Notes: e.Notes,
			})
		}
		out = append(out, block)
	}
	return out
}

// copyPlan turns a loaded training's sessions and schedule into new ones (without ids) for
// insertSessions and writeSchedule.
func copyPlan(t Training) ([]SessionInput, [][]SlotInput) {
	sessions := make([]SessionInput, 0, len(t.Sessions))
	index := map[uuid.UUID]int{}
	for i, s := range t.Sessions {
		sessions = append(sessions, SessionInput{Name: s.Name, Notes: s.Notes, Blocks: copyBlocks(s.Blocks)})
		index[s.ID] = i
	}
	schedule := make([][]SlotInput, 0, len(t.Schedule))
	for _, week := range t.Schedule {
		slots := make([]SlotInput, 0, len(week))
		for _, slot := range week {
			slots = append(slots, SlotInput{Session: index[slot.SessionID]})
		}
		schedule = append(schedule, slots)
	}
	return sessions, schedule
}
