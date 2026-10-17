package training

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Create validates a new training and saves it with its whole tree in one transaction.
// Saving it active deactivates the client's other trainings.
func (s *Service) Create(ctx context.Context, in Input) (repos.CreateTrainingRow, error) {
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return repos.CreateTrainingRow{}, err
	}
	if err := validate(&in, lk); err != nil {
		return repos.CreateTrainingRow{}, err
	}
	var training repos.CreateTrainingRow
	err = s.inTx(ctx, func(q Queries) error {
		if err := checkClientExists(ctx, q, in.ClientID); err != nil {
			return err
		}
		// Deactivate the client's trainings first, or the one-active-per-client index rejects
		// the new active one.
		if in.Active {
			if err := q.DeactivateClientTrainings(ctx, in.ClientID); err != nil {
				return err
			}
		}
		var err error
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: in.ClientID, Name: in.Name, Description: in.Description, Active: in.Active,
		})
		if err != nil {
			return err
		}
		return insertUnits(ctx, q, training.ID, in.Units)
	})
	if err != nil {
		return repos.CreateTrainingRow{}, mapError(err)
	}
	return training, nil
}
