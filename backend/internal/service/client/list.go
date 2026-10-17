package client

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// List returns all live clients with their training count and active training's name.
func (s *Service) List(ctx context.Context) ([]repos.ListUsersRow, error) {
	return s.q.ListUsers(ctx)
}
