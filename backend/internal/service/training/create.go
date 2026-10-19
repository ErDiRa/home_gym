package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Create validates a new training and saves it for the user with its whole tree in one
// transaction.
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
		var err error
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: userID, Name: in.Name, Description: in.Description, Active: in.Active,
			StartDate: in.start, Weeks: in.Weeks,
		})
		if err != nil {
			return err
		}
		ids, err := insertSessions(ctx, q, userID, training.ID, in.Sessions)
		if err != nil {
			return err
		}
		return writeSchedule(ctx, q, training.ID, in.Schedule, ids)
	})
	if err != nil {
		return repos.CreateTrainingRow{}, mapError(err)
	}
	return training, nil
}
