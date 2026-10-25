package whoop

import (
	"context"
	"errors"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Configured reports whether the WHOOP app is set up, without which nobody can connect.
func (s *Service) Configured() bool {
	return s.cfg.ClientID != "" && s.cfg.ClientSecret != "" && s.cfg.RedirectURL != ""
}

// AuthURL is the WHOOP page where the user allows HomeGym to read their data; WHOOP then
// redirects to the redirect URL with a code and the given state.
func (s *Service) AuthURL(state string) string {
	return s.baseURL + "/oauth/oauth2/auth?" + url.Values{
		"response_type": {"code"},
		"client_id":     {s.cfg.ClientID},
		"redirect_uri":  {s.cfg.RedirectURL},
		"scope":         {"read:workout read:recovery offline"},
		"state":         {state},
	}.Encode()
}

// Connect swaps the code WHOOP sent back for the user's tokens and stores them.
func (s *Service) Connect(ctx context.Context, userID uuid.UUID, code string) error {
	_, err := s.token(ctx, userID, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {s.cfg.RedirectURL},
	})
	return err
}

// Connected reports whether the user has connected WHOOP.
func (s *Service) Connected(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, err := s.q.GetConnection(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Disconnect forgets the user's tokens; the data of linked workouts stays.
func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	return s.q.DeleteConnection(ctx, userID)
}
