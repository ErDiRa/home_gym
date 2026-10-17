package client

import (
	"context"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/client/repos"
)

// Create adds a client with an initial password (stored as a bcrypt hash).
func (s *Service) Create(ctx context.Context, in Input) (repos.CreateUserRow, error) {
	if err := validate(&in); err != nil {
		return repos.CreateUserRow{}, err
	}
	// bcrypt only uses the first 72 bytes.
	if utf8.RuneCountInString(in.Password) < 8 || len(in.Password) > 72 {
		return repos.CreateUserRow{}, service.Invalid("password must be 8 to 72 characters")
	}
	if err := s.checkNotAdminEmail(ctx, in.Email); err != nil {
		return repos.CreateUserRow{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.hashCost)
	if err != nil {
		return repos.CreateUserRow{}, err
	}
	client, err := s.q.CreateUser(ctx, repos.CreateUserParams{
		Email:        in.Email,
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		PasswordHash: string(hash),
	})
	if err != nil {
		return repos.CreateUserRow{}, mapError(err)
	}
	return client, nil
}
