package training

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Active returns the user's active training, the one the tracker shows, with its whole tree;
// nil when none is active.
func (s *Service) Active(ctx context.Context, userID uuid.UUID) (*Training, error) {
	id, err := s.q.GetActiveTrainingID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := load(ctx, s.q, userID, id)
	if err != nil {
		return nil, mapError(err)
	}
	return &t, nil
}
