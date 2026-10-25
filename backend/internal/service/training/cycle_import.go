package training

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// maxImportTrainings bounds the trainings of a cycle import.
const maxImportTrainings = 60

// CycleImportInput is a whole cycle with its trainings, as a cycle import sends it. The client
// has resolved the names of the import file (exercises, units) to ids, so each training is an
// Input like Create takes; their cycle is the new one, whatever they say.
type CycleImportInput struct {
	Cycle     cycle.Input `json:"cycle"`
	Trainings []Input     `json:"trainings"`
}

// ImportedCycle is the cycle a cycle import created, with its trainings in the order sent; the
// embedded row's fields sit at the top level.
type ImportedCycle struct {
	repos.CreateCycleRow
	Trainings []repos.CreateTrainingRow `json:"trainings"`
}

// ImportCycle validates a cycle and all its trainings, then saves them in one transaction:
// either the whole cycle is imported or nothing is. It lives here rather than in the cycle
// service because writing the trainings is this service's job.
func (s *Service) ImportCycle(ctx context.Context, userID uuid.UUID, in CycleImportInput) (ImportedCycle, error) {
	if err := cycle.Validate(&in.Cycle); err != nil {
		return ImportedCycle{}, err
	}
	if len(in.Trainings) > maxImportTrainings {
		return ImportedCycle{}, service.Invalid(fmt.Sprintf("a cycle import holds at most %d trainings", maxImportTrainings))
	}
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return ImportedCycle{}, err
	}
	for i := range in.Trainings {
		t := &in.Trainings[i]
		if err := validate(t, lk); err != nil {
			return ImportedCycle{}, inTraining(err, i, t, in.Cycle)
		}
	}

	imported := ImportedCycle{Trainings: make([]repos.CreateTrainingRow, 0, len(in.Trainings))}
	err = s.inTx(ctx, func(q Queries) error {
		var err error
		imported.CreateCycleRow, err = q.CreateCycle(ctx, repos.CreateCycleParams{
			UserID: userID, Name: in.Cycle.Name, Description: in.Cycle.Description,
			StartDate: in.Cycle.Start, Weeks: in.Cycle.Weeks,
		})
		if err != nil {
			return err
		}
		for i, g := range in.Cycle.Goals {
			err := q.InsertCycleGoal(ctx, repos.InsertCycleGoalParams{
				CycleID: imported.ID, Position: int32(i + 1), Title: g.Title, Target: g.Target, Status: g.Status, Notes: g.Notes,
			})
			if err != nil {
				return err
			}
		}
		for i, t := range in.Trainings {
			training, err := q.CreateTraining(ctx, repos.CreateTrainingParams{
				UserID: userID, CycleID: &imported.ID, Name: t.Name, Description: t.Description, Active: t.Active,
				StartDate: t.start, Weeks: t.Weeks,
			})
			if err != nil {
				return err
			}
			ids, err := insertSessions(ctx, q, userID, training.ID, t.Sessions)
			if err == nil {
				err = writeSchedule(ctx, q, training.ID, t.Schedule, ids)
			}
			if err != nil {
				return inTraining(mapError(err), i, &t, in.Cycle)
			}
			imported.Trainings = append(imported.Trainings, training)
		}
		return nil
	})
	if err != nil {
		return ImportedCycle{}, mapError(err)
	}
	return imported, nil
}

// inTraining prefixes a training's validation error with where it is, e.g.
// `Week 3 "Strength - Week 3": Session 2, block B: …`, so the import can show it. A training whose
// start couldn't be read is named by its position instead.
func inTraining(err error, i int, t *Input, c cycle.Input) error {
	var svcErr *service.Error
	if !errors.As(err, &svcErr) || svcErr.Kind != service.KindInvalid {
		return err
	}
	where := fmt.Sprintf("Training %d", i+1)
	if days := int(t.start.Time.Sub(c.Start.Time).Hours() / 24); t.start.Valid && days >= 0 {
		where = fmt.Sprintf("Week %d", days/7+1)
	}
	return service.InvalidCause(fmt.Sprintf("%s %q: %s", where, t.Name, svcErr.Msg), svcErr.Err)
}
