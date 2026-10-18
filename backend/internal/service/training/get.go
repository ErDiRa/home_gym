package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Get returns one of the user's trainings with its units, blocks and exercises in order.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Training, error) {
	t, err := load(ctx, s.q, userID, id)
	return t, mapError(err)
}

// load reads one of the user's trainings with its units, blocks and exercises in order.
func load(ctx context.Context, q Queries, userID, id uuid.UUID) (Training, error) {
	training, err := q.GetTraining(ctx, repos.GetTrainingParams{ID: id, UserID: userID})
	if err != nil {
		return Training{}, err
	}
	units, err := q.ListTrainingUnits(ctx, id)
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

	// The lists come back ordered by position within their parent, so appending keeps order.
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
	blocksByUnit := map[uuid.UUID][]Block{}
	for _, b := range blocks {
		blocksByUnit[b.UnitID] = append(blocksByUnit[b.UnitID], Block{
			ID: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets, Notes: b.Notes,
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
	resp := Training{GetTrainingRow: training, Units: make([]Unit, 0, len(units))}
	for _, u := range units {
		resp.Units = append(resp.Units, Unit{ID: u.ID, Name: u.Name, Notes: u.Notes, Blocks: orEmpty(blocksByUnit[u.ID])})
	}
	return resp, nil
}
