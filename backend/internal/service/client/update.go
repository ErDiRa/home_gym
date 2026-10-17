package client

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// Update changes a client's name and email. The password can't be changed here.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in Input) (repos.UpdateUserRow, error) {
	if err := validate(&in); err != nil {
		return repos.UpdateUserRow{}, err
	}
	if err := s.checkNotAdminEmail(ctx, in.Email); err != nil {
		return repos.UpdateUserRow{}, err
	}
	client, err := s.q.UpdateUser(ctx, repos.UpdateUserParams{
		ID:        id,
		Email:     in.Email,
		FirstName: in.FirstName,
		LastName:  in.LastName,
	})
	if err != nil {
		return repos.UpdateUserRow{}, mapError(err)
	}
	return client, nil
}
