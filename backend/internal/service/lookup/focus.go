package lookup

import (
	"context"
	"strings"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

// FocusInput is a new focus, e.g. "Hip mobility" with the alias "HIP".
type FocusInput struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// CreateFocus adds a focus for exercises.
func (s *Service) CreateFocus(ctx context.Context, in FocusInput) (repos.CreateFocusRow, error) {
	name, alias := strings.TrimSpace(in.Name), strings.TrimSpace(in.Alias)
	if name == "" || tooLong(name, 60) {
		return repos.CreateFocusRow{}, service.Invalid("name is required (max 60 characters)")
	}
	if alias == "" || tooLong(alias, 10) {
		return repos.CreateFocusRow{}, service.Invalid("alias is required (max 10 characters)")
	}
	focus, err := s.q.CreateFocus(ctx, repos.CreateFocusParams{Name: name, Alias: alias})
	if isUniqueViolation(err) {
		return repos.CreateFocusRow{}, service.Conflict("this focus already exists", err)
	}
	return focus, err
}
