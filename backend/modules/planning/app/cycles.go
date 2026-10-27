package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// CreateCycle validates a new cycle and saves it for the user with its goals. It starts without
// trainings.
func (s *Service) CreateCycle(ctx context.Context, userID uuid.UUID, in CycleInput) (uuid.UUID, error) {
	if err := validateCycle(&in); err != nil {
		return uuid.Nil, err
	}
	return s.cycles.CreateCycle(ctx, userID, in)
}

// UpdateCycle validates a cycle and replaces its fields and goals. Moving its period doesn't move
// its trainings.
func (s *Service) UpdateCycle(ctx context.Context, userID, id uuid.UUID, in CycleInput) error {
	if err := validateCycle(&in); err != nil {
		return err
	}
	return cycleError(s.cycles.UpdateCycle(ctx, userID, id, in))
}

// DeleteCycle soft-deletes one of the user's cycles. Its trainings are soft-deleted with it when
// withTrainings is set, else they stay and just leave the cycle.
func (s *Service) DeleteCycle(ctx context.Context, userID, id uuid.UUID, withTrainings bool) error {
	return cycleError(s.cycles.DeleteCycle(ctx, userID, id, withTrainings))
}

// maxImportTrainings bounds the trainings of a cycle import.
const maxImportTrainings = 60

// ImportCycle validates a cycle and all its trainings, then saves them in one transaction: either
// the whole cycle is imported or nothing is. The client has resolved the names of the import file
// (exercises, units) to ids; the trainings' cycle is the new one, whatever they say.
func (s *Service) ImportCycle(ctx context.Context, userID uuid.UUID, cycle CycleInput, trainings []Input) (ImportedCycle, error) {
	if err := validateCycle(&cycle); err != nil {
		return ImportedCycle{}, err
	}
	if len(trainings) > maxImportTrainings {
		return ImportedCycle{}, errs.Invalid(fmt.Sprintf("a cycle import holds at most %d trainings", maxImportTrainings))
	}
	lk, err := s.lookups(ctx)
	if err != nil {
		return ImportedCycle{}, err
	}
	for i := range trainings {
		t := &trainings[i]
		if err := validate(t, lk); err != nil {
			return ImportedCycle{}, inTraining(err, i, t, cycle)
		}
		if err := s.checkExercises(ctx, userID, t.Sessions...); err != nil {
			return ImportedCycle{}, inTraining(err, i, t, cycle)
		}
	}
	imported, err := s.cycles.ImportCycle(ctx, userID, cycle, trainings)
	return imported, trainingError(err)
}

// inTraining prefixes a training's validation error with where it is, e.g.
// `Week 3 "Strength - Week 3": Session 2, block B: …`, so the import can show it. A training whose
// start couldn't be read is named by its position instead.
func inTraining(err error, i int, t *Input, c CycleInput) error {
	var e *errs.Error
	if !errors.As(err, &e) || e.Kind != errs.KindInvalid {
		return err
	}
	where := fmt.Sprintf("Training %d", i+1)
	if days := int(t.start.Sub(c.start).Hours() / 24); !t.start.IsZero() && days >= 0 {
		where = fmt.Sprintf("Week %d", days/7+1)
	}
	return errs.InvalidCause(fmt.Sprintf("%s %q: %s", where, t.Name, e.Msg), e.Err)
}

func cycleError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("cycle not found", err)
	}
	return err
}
