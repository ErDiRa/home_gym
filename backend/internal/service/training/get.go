package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Get returns one of the user's trainings with its sessions, blocks, exercises and schedule.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Training, error) {
	t, err := load(ctx, s.q, userID, id)
	return t, mapError(err)
}

// load reads one of the user's trainings with its sessions, blocks and exercises in order, and
// its schedule with one list per week.
func load(ctx context.Context, q Queries, userID, id uuid.UUID) (Training, error) {
	training, err := q.GetTraining(ctx, repos.GetTrainingParams{ID: id, UserID: userID})
	if err != nil {
		return Training{}, err
	}
	sessions, err := q.ListTrainingSessions(ctx, id)
	if err != nil {
		return Training{}, err
	}
	blocks, err := q.ListTrainingBlocks(ctx, id)
	if err != nil {
		return Training{}, err
	}
	exercises, err := q.ListTrainingExercises(ctx, id)
	if err != nil {
		return Training{}, err
	}
	slots, err := q.ListWeekSessions(ctx, id)
	if err != nil {
		return Training{}, err
	}

	blocksBySession := groupBlocks(blocks, exercises)
	resp := Training{GetTrainingRow: training, Sessions: make([]Session, 0, len(sessions))}
	for _, s := range sessions {
		resp.Sessions = append(resp.Sessions, Session{ID: s.ID, Name: s.Name, Notes: s.Notes, Blocks: orEmpty(blocksBySession[s.ID])})
	}
	resp.Schedule = make([][]Slot, training.Weeks)
	for i := range resp.Schedule {
		resp.Schedule[i] = []Slot{}
	}
	for _, s := range slots {
		// Slots of weeks past the end (only possible by hand in the database) are left out.
		if s.Week <= training.Weeks {
			resp.Schedule[s.Week-1] = append(resp.Schedule[s.Week-1], Slot{ID: s.ID, SessionID: s.SessionID})
		}
	}
	return resp, nil
}

// groupBlocks builds the blocks with their exercises, grouped by session. The rows come back
// ordered by position within their parent, so appending keeps the order.
func groupBlocks(blocks []repos.ListTrainingBlocksRow, exercises []repos.ListTrainingExercisesRow) map[uuid.UUID][]Block {
	exercisesByBlock := map[uuid.UUID][]Exercise{}
	for _, e := range exercises {
		exercisesByBlock[e.BlockID] = append(exercisesByBlock[e.BlockID], Exercise{
			ExerciseID: e.ExerciseID, Name: e.Name, Type: e.Type, Sets: e.Sets, Tempo: e.Tempo, Notes: e.Notes,
			AmountWithUnits: AmountWithUnits{
				Amount: Amount{
					RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
					LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit,
				},
				RepsUnitName: e.RepsUnitName, LoadUnitName: e.LoadUnitName,
			},
		})
	}
	blocksBySession := map[uuid.UUID][]Block{}
	for _, b := range blocks {
		blocksBySession[b.SessionID] = append(blocksBySession[b.SessionID], Block{
			ID: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets, Notes: b.Notes,
			Score: Score{ScoreType: b.ScoreType, TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds},
			AmountWithUnits: AmountWithUnits{
				Amount: Amount{
					RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
					LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit,
				},
				RepsUnitName: b.RepsUnitName, LoadUnitName: b.LoadUnitName,
			},
			Exercises: orEmpty(exercisesByBlock[b.ID]),
		})
	}
	return blocksBySession
}
