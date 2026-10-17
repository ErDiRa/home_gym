package client

import (
	"context"

	"github.com/google/uuid"
)

// Get returns a client with their trainings, the active one first.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Client, error) {
	client, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return Client{}, mapError(err)
	}
	trainings, err := s.q.ListClientTrainings(ctx, id)
	if err != nil {
		return Client{}, err
	}
	return Client{GetUserByIDRow: client, Trainings: trainings}, nil
}
