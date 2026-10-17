package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Update validates a training and replaces its fields and tree in one transaction; it can
// also move the training to another client. Units sent with an id keep it.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in Input) (repos.UpdateTrainingRow, error) {
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return repos.UpdateTrainingRow{}, err
	}
	if err := validate(&in, lk); err != nil {
		return repos.UpdateTrainingRow{}, err
	}
	var training repos.UpdateTrainingRow
	err = s.inTx(ctx, func(q Queries) error {
		if err := checkClientExists(ctx, q, in.ClientID); err != nil {
			return err
		}
		// Deactivate all the client's trainings, this one included; the update below sets
		// this one active again.
		if in.Active {
			if err := q.DeactivateClientTrainings(ctx, in.ClientID); err != nil {
				return err
			}
		}
		var err error
		training, err = q.UpdateTraining(ctx, repos.UpdateTrainingParams{
			ID: id, UserID: in.ClientID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.start, EndDate: in.end,
		})
		if err != nil {
			return err
		}
		return updateUnits(ctx, q, id, in.Units)
	})
	if err != nil {
		return repos.UpdateTrainingRow{}, mapError(err)
	}
	return training, nil
}
