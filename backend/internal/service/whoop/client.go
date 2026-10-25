package whoop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

// tokenResponse is what the WHOOP token endpoint answers.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// token posts form (a code exchange or a refresh) to the WHOOP token endpoint and stores the
// tokens it returns for the user.
func (s *Service) token(ctx context.Context, userID uuid.UUID, form url.Values) (string, error) {
	form.Set("client_id", s.cfg.ClientID)
	form.Set("client_secret", s.cfg.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/oauth/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var tok tokenResponse
	if err := s.do(req, &tok); err != nil {
		return "", err
	}
	err = s.q.SaveConnection(ctx, repos.SaveConnectionParams{
		UserID: userID, AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second), Valid: true},
	})
	return tok.AccessToken, err
}

// accessToken returns a valid access token of the user, refreshing it when it is about to
// expire. The refresh token rotates, so the new one is stored right away.
func (s *Service) accessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	c, err := s.q.GetConnection(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errNotConnected
	}
	if err != nil {
		return "", err
	}
	if time.Until(c.ExpiresAt.Time) > time.Minute {
		return c.AccessToken, nil
	}
	return s.token(ctx, userID, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {c.RefreshToken}, "scope": {"offline"},
	})
}

// get calls a WHOOP API path as the user and decodes the answer into dst.
func (s *Service) get(ctx context.Context, userID uuid.UUID, path string, query url.Values, dst any) error {
	token, err := s.accessToken(ctx, userID)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return s.do(req, dst)
}

// do sends req and decodes a 2xx JSON answer into dst. WHOOP rejecting the credentials asks
// the user to connect again; any other failure is an internal error.
func (s *Service) do(req *http.Request, dst any) error {
	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("whoop %s: %w", req.URL.Path, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusBadRequest && strings.HasSuffix(req.URL.Path, "/token"):
		return errReconnect
	case res.StatusCode == http.StatusNotFound:
		return notFoundActivity
	case res.StatusCode >= 300:
		return fmt.Errorf("whoop %s: status %d", req.URL.Path, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(dst)
}
