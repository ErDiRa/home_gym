package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// insertUnits writes the units, blocks and exercises of a new training, in input order.
// Unit ids in the input are ignored: a new training has no units yet.
func insertUnits(ctx context.Context, q Queries, trainingID uuid.UUID, units []UnitInput) error {
	for i, u := range units {
		unitID, err := q.InsertTrainingUnit(ctx, repos.InsertTrainingUnitParams{
			TrainingID: trainingID, Position: int32(i + 1), Name: u.Name, Notes: u.Notes,
			Weekdays: u.Weekdays, Dates: u.dates,
		})
		if err != nil {
			return err
		}
		if err := insertBlocks(ctx, q, unitID, u.Blocks); err != nil {
			return err
		}
	}
	return nil
}

// updateUnits replaces the units of an existing training: units missing from the input are
// deleted, units with an id are updated in place (keeping their id), units without one are
// inserted. All blocks and exercises are deleted and re-inserted.
func updateUnits(ctx context.Context, q Queries, trainingID uuid.UUID, units []UnitInput) error {
	keep := []uuid.UUID{}
	for _, u := range units {
		if u.ID != nil {
			keep = append(keep, *u.ID)
		}
	}
	if err := q.DeleteTrainingUnitsExcept(ctx, repos.DeleteTrainingUnitsExceptParams{TrainingID: trainingID, KeepIds: keep}); err != nil {
		return err
	}
	// Blocks cascade to their exercises; both are re-inserted below in input order.
	if err := q.DeleteTrainingBlocks(ctx, trainingID); err != nil {
		return err
	}

	for i, u := range units {
		position := int32(i + 1)
		var unitID uuid.UUID
		if u.ID != nil {
			n, err := q.UpdateTrainingUnit(ctx, repos.UpdateTrainingUnitParams{
				ID: *u.ID, TrainingID: trainingID, Position: position, Name: u.Name, Notes: u.Notes,
				Weekdays: u.Weekdays, Dates: u.dates,
			})
			if err != nil {
				return err
			}
			// The id belongs to another training (or none): refuse rather than move it.
			if n == 0 {
				return service.Invalid("unknown unit")
			}
			unitID = *u.ID
		} else {
			id, err := q.InsertTrainingUnit(ctx, repos.InsertTrainingUnitParams{
				TrainingID: trainingID, Position: position, Name: u.Name, Notes: u.Notes,
				Weekdays: u.Weekdays, Dates: u.dates,
			})
			if err != nil {
				return err
			}
			unitID = id
		}
		if err := insertBlocks(ctx, q, unitID, u.Blocks); err != nil {
			return err
		}
	}
	return nil
}

// insertBlocks inserts a unit's blocks and their exercises, in input order.
func insertBlocks(ctx context.Context, q Queries, unitID uuid.UUID, blocks []BlockInput) error {
	for j, b := range blocks {
		blockID, err := q.InsertTrainingBlock(ctx, repos.InsertTrainingBlockParams{
			UnitID: unitID, Position: int32(j + 1), Kind: b.Kind, Name: b.Name, Sets: b.Sets,
			RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
			LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, Notes: b.Notes,
		})
		if err != nil {
			return err
		}
		for k, e := range b.Exercises {
			err := q.InsertTrainingExercise(ctx, repos.InsertTrainingExerciseParams{
				BlockID: blockID, ExerciseID: e.ExerciseID, Position: int32(k + 1), Sets: e.Sets,
				RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
				LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, Tempo: e.Tempo, Notes: e.Notes,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// copyUnits turns loaded units into new ones (without ids) for insertUnits.
func copyUnits(units []Unit) []UnitInput {
	out := make([]UnitInput, 0, len(units))
	for _, u := range units {
		unit := UnitInput{Name: u.Name, Notes: u.Notes, Weekdays: u.Weekdays}
		unit.dates, _ = parseDates(u.Dates) // formatted by load, so always valid
		for _, b := range u.Blocks {
			block := BlockInput{Kind: b.Kind, Name: b.Name, Sets: b.Sets, Amount: b.Amount, Notes: b.Notes}
			for _, e := range b.Exercises {
				block.Exercises = append(block.Exercises, ExerciseInput{
					ExerciseID: e.ExerciseID, Sets: e.Sets, Amount: e.Amount, Tempo: e.Tempo, Notes: e.Notes,
				})
			}
			unit.Blocks = append(unit.Blocks, block)
		}
		out = append(out, unit)
	}
	return out
}
