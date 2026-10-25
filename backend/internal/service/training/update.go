package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Update validates a training and replaces its fields, cycle, tree and schedule in one transaction.
// Sessions and slots sent with an id keep it, so the workouts done of them stay linked.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, in Input) (repos.UpdateTrainingRow, error) {
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return repos.UpdateTrainingRow{}, err
	}
	if err := validate(&in, lk); err != nil {
		return repos.UpdateTrainingRow{}, err
	}
	var training repos.UpdateTrainingRow
	err = s.inTx(ctx, func(q Queries) error {
		if err := checkCycle(ctx, q, userID, in.CycleID); err != nil {
			return err
		}
		var err error
		training, err = q.UpdateTraining(ctx, repos.UpdateTrainingParams{
			ID: id, UserID: userID, CycleID: in.CycleID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.start, Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		ids, err := updateSessions(ctx, q, userID, id, in.Sessions)
		if err != nil {
			return err
		}
		return writeSchedule(ctx, q, id, in.Schedule, ids)
	})
	if err != nil {
		return repos.UpdateTrainingRow{}, mapError(err)
	}
	return training, nil
}
