package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Duplicate copies a training, with its units, blocks and exercises, for a client (the same
// or another one). The copy starts inactive.
func (s *Service) Duplicate(ctx context.Context, id, clientID uuid.UUID) (repos.CreateTrainingRow, error) {
	// The source was validated when it was saved, so the copy isn't checked again.
	var training repos.CreateTrainingRow
	err := s.inTx(ctx, func(q Queries) error {
		source, err := load(ctx, q, id)
		if err != nil {
			return err
		}
		if err := checkClientExists(ctx, q, clientID); err != nil {
			return err
		}
		training, err = q.CreateTraining(ctx, repos.CreateTrainingParams{
			UserID: clientID, Name: source.Name, Description: source.Description,
		})
		if err != nil {
			return err
		}
		return insertUnits(ctx, q, training.ID, copyUnits(source.Units))
	})
	if err != nil {
		return repos.CreateTrainingRow{}, mapError(err)
	}
	return training, nil
}
