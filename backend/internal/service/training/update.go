package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Update validates a training and replaces its fields and tree in one transaction. Units sent
// with an id keep it, so the workouts done of them stay linked.
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
		// Deactivate all the user's trainings, this one included; the update below sets this
		// one active again.
		if in.Active {
			if err := q.DeactivateUserTrainings(ctx, userID); err != nil {
				return err
			}
		}
		var err error
		training, err = q.UpdateTraining(ctx, repos.UpdateTrainingParams{
			ID: id, UserID: userID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.start, EndDate: in.end, SessionsPerWeek: in.SessionsPerWeek,
		})
		if err != nil {
			return err
		}
		return updateUnits(ctx, q, userID, id, in.Units)
	})
	if err != nil {
		return repos.UpdateTrainingRow{}, mapError(err)
	}
	return training, nil
}
