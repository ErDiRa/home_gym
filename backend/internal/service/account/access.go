package account

import (
	"context"

	"github.com/google/uuid"
)

// IsAdmin reports whether id is a live admin; auth.RequireAdmin checks it on every
// /api/admin request.
func (s *Service) IsAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.q.IsAdmin(ctx, id)
}

// UserExists reports whether id is a live user; auth.RequireUser checks it on every
// /api/client request.
func (s *Service) UserExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.q.UserExists(ctx, id)
}
