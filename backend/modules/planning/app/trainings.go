package app

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// CreateTraining validates a new training and saves it for the user with its whole tree. Its
// exercises must be the user's, its cycle, if any, one of the user's.
func (s *Service) CreateTraining(ctx context.Context, userID uuid.UUID, in Input) (TrainingHeader, error) {
	if err := s.validateTraining(ctx, userID, &in); err != nil {
		return TrainingHeader{}, err
	}
	t, err := s.trainings.CreateTraining(ctx, userID, in)
	return t, trainingError(err)
}

// UpdateTraining validates a training and replaces its fields, cycle, tree and schedule. Sessions
// and slots sent with an id keep it, so the workouts done of them stay linked; those ids must be
// the training's own.
func (s *Service) UpdateTraining(ctx context.Context, userID, id uuid.UUID, in Input) (TrainingHeader, error) {
	if err := s.validateTraining(ctx, userID, &in); err != nil {
		return TrainingHeader{}, err
	}
	t, err := s.trainings.UpdateTraining(ctx, userID, id, func(_ context.Context, current Training) (Input, error) {
		return in, checkKeptIDs(current, in)
	})
	return t, trainingError(err)
}

// checkKeptIDs fails unless every session and slot id of in is one of the current training's: an
// id of another training (or of none) is refused rather than moved.
func checkKeptIDs(current Training, in Input) error {
	sessions, slots := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
	for _, s := range current.Sessions {
		sessions[s.ID] = true
	}
	for _, week := range current.Schedule {
		for _, slot := range week {
			slots[slot.ID] = true
		}
	}
	for _, s := range in.Sessions {
		if s.ID != nil && !sessions[*s.ID] {
			return errs.Invalid("unknown session")
		}
	}
	for _, week := range in.Schedule {
		for _, slot := range week {
			if slot.ID != nil && !slots[*slot.ID] {
				return errs.Invalid("unknown slot in the schedule")
			}
		}
	}
	return nil
}

// DuplicateTraining copies one of the user's trainings, with its sessions and schedule. The copy
// starts inactive, and as the next phase: as many weeks, from the Monday after the source ends, in
// the same cycle (e.g. next week of a plan that arrives week by week). The source was validated
// when it was saved, so the copy isn't checked again.
func (s *Service) DuplicateTraining(ctx context.Context, userID, id uuid.UUID) (TrainingHeader, error) {
	source, err := s.trainings.Training(ctx, userID, id)
	if err != nil {
		return TrainingHeader{}, trainingError(err)
	}
	sessions, schedule := copyPlan(source)
	t, err := s.trainings.CreateTraining(ctx, userID, Input{
		CycleID: source.CycleID, Name: source.Name, Description: source.Description,
		Weeks: source.Weeks, Sessions: sessions, Schedule: schedule,
		start: source.EndDate.AddDate(0, 0, 1), // the end is a Sunday
	})
	return t, trainingError(err)
}

// DeleteTraining soft-deletes one of the user's trainings.
func (s *Service) DeleteTraining(ctx context.Context, userID, id uuid.UUID) error {
	return trainingError(s.trainings.DeleteTraining(ctx, userID, id))
}

// validateTraining validates a training against the catalog's kinds, units and the user's
// exercises, in place.
func (s *Service) validateTraining(ctx context.Context, userID uuid.UUID, in *Input) error {
	lk, err := s.lookups(ctx)
	if err != nil {
		return err
	}
	if err := validate(in, lk); err != nil {
		return err
	}
	return s.checkExercises(ctx, userID, in.Sessions...)
}

// copyPlan turns a saved training's sessions and schedule into new ones (without ids).
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

// copyBlocks turns saved blocks into new ones.
func copyBlocks(blocks []Block) []BlockInput {
	out := make([]BlockInput, 0, len(blocks))
	for _, b := range blocks {
		block := BlockInput{Kind: b.Kind, Name: b.Name, Sets: b.Sets, Amount: b.Amount, Score: b.Score, Notes: b.Notes}
		for _, e := range b.Exercises {
			block.Exercises = append(block.Exercises, ExerciseInput{
				ExerciseID: e.ExerciseID, Sets: e.Sets, Amount: e.Amount, Tempo: e.Tempo, Notes: e.Notes,
			})
		}
		out = append(out, block)
	}
	return out
}

// trainingError turns a repository's error into the one the caller sees; an *errs.Error (e.g.
// from the update's check) passes through.
func trainingError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("training not found", err)
	case errors.Is(err, ErrUnknownCycle):
		return errs.InvalidCause("unknown cycle", err)
	case errors.Is(err, ErrUnknownReference):
		// Kinds, units and exercises are checked before; this is an exercise deleted meanwhile.
		return errs.InvalidCause("unknown exercise", err)
	default:
		return err
	}
}
