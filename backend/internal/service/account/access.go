package account

import (
	"context"

	"github.com/google/uuid"
)

// UserExists reports whether id is a live user; auth.RequireUser checks it on every /api
// request.
func (s *Service) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.q.UserExists(ctx, id)
}
