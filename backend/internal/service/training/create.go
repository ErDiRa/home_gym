package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Create validates a new training and saves it for the user with its whole tree in one
// transaction. Saving it active deactivates the user's other trainings.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (repos.CreateTrainingRow, error) {
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return repos.CreateTrainingRow{}, err
	}
	if err := validate(&in, lk); err != nil {
		return repos.CreateTrainingRow{}, err
	}
	var training repos.CreateTrainingRow
	err = s.inTx(ctx, func(q Queries) error {
		// Deactivate the user's trainings first, or the one-active-per-user index rejects the
		// new active one.
		if in.Active {
			if err := q.DeactivateUserTrainings(ctx, userID); err != nil {
				return err
			}
		}
		var err error
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: userID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.start, EndDate: in.end, SessionsPerWeek: in.SessionsPerWeek,
		})
		if err != nil {
			return err
		}
		return insertUnits(ctx, q, userID, training.ID, in.Units)
	})
	if err != nil {
		return repos.CreateTrainingRow{}, mapError(err)
	}
	return training, nil
}
