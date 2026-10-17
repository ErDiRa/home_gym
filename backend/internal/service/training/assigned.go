package training

import (
	"context"

	"github.com/google/uuid"
)

// Assigned returns the trainings a client follows, with their whole tree: their active one,
// or none.
func (s *Service) Assigned(ctx context.Context, clientID uuid.UUID) ([]Training, error) {
	ids, err := s.q.ListAssignedTrainingIDs(ctx, clientID)
	if err != nil {
		return nil, err
	}
	trainings := make([]Training, 0, len(ids))
	for _, id := range ids {
		t, err := load(ctx, s.q, id)
		if err != nil {
			return nil, mapError(err)
		}
		trainings = append(trainings, t)
	}
	return trainings, nil
}
