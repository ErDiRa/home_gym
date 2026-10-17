package client

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
)

// Delete soft-deletes a client; they can no longer sign in, their open sessions are rejected
// by auth.RequireUser, and their trainings are hidden.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("client not found", nil)
	}
	return nil
}
