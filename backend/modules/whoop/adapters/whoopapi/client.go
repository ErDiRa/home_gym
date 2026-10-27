// Package whoopapi is the client of the WHOOP API (v2): OAuth tokens, workouts and recoveries.
package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/whoop/app"
)

// Config is the app registered in the WHOOP Developer Dashboard.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string // where WHOOP sends the user back to: <origin>/api/whoop/callback
}

// Client implements app.API.
type Client struct {
	cfg     Config
	http    *http.Client
	baseURL string // the WHOOP API; a test server in tests
}

var _ app.API = (*Client)(nil)

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}, baseURL: "https://api.prod.whoop.com"}
}

func (c *Client) Configured() bool {
	return c.cfg.ClientID != "" && c.cfg.ClientSecret != "" && c.cfg.RedirectURL != ""
}

func (c *Client) AuthURL(state string) string {
	return c.baseURL + "/oauth/oauth2/auth?" + url.Values{
		"response_type": {"code"},
		"client_id":     {c.cfg.ClientID},
		"redirect_uri":  {c.cfg.RedirectURL},
		"scope":         {"read:workout read:recovery offline"},
		"state":         {state},
	}.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (app.Tokens, error) {
	return c.token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {c.cfg.RedirectURL}})
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (app.Tokens, error) {
	return c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "scope": {"offline"}})
}

// token posts form (a code exchange or a refresh) to the token endpoint.
func (c *Client) token(ctx context.Context, form url.Values) (app.Tokens, error) {
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return app.Tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
	}
	if err := c.do(req, &tok); err != nil {
		return app.Tokens{}, err
	}
	return app.Tokens{
		AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
	}, nil
}

// activity is a WHOOP workout as the API returns it (only the fields used).
type activity struct {
	ID         uuid.UUID `json:"id"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	SportName  string    `json:"sport_name"`
	ScoreState string    `json:"score_state"`
	Score      *struct {
		Strain           float64 `json:"strain"`
		AverageHeartRate int32   `json:"average_heart_rate"`
		MaxHeartRate     int32   `json:"max_heart_rate"`
		Kilojoule        float64 `json:"kilojoule"`
		ZoneDurations    struct {
			Zero  int32 `json:"zone_zero_milli"`
			One   int32 `json:"zone_one_milli"`
			Two   int32 `json:"zone_two_milli"`
			Three int32 `json:"zone_three_milli"`
			Four  int32 `json:"zone_four_milli"`
			Five  int32 `json:"zone_five_milli"`
		} `json:"zone_durations"`
	} `json:"score"`
}

func (a activity) toApp() app.Activity {
	out := app.Activity{ID: a.ID, SportName: a.SportName, Start: a.Start, End: a.End, ScoreState: a.ScoreState}
	if sc := a.Score; sc != nil {
		z := sc.ZoneDurations
		out.Strain, out.AvgHR, out.MaxHR, out.Kilojoule = &sc.Strain, &sc.AverageHeartRate, &sc.MaxHeartRate, &sc.Kilojoule
		out.ZoneMs = []int32{z.Zero, z.One, z.Two, z.Three, z.Four, z.Five}
	}
	return out
}

func (c *Client) Activities(ctx context.Context, accessToken string, from, to time.Time) ([]app.Activity, error) {
	var page struct {
		Records []activity `json:"records"`
	}
	query := url.Values{"start": {from.Format(time.RFC3339)}, "end": {to.Format(time.RFC3339)}, "limit": {"25"}}
	if err := c.get(ctx, accessToken, "/developer/v2/activity/workout", query, &page); err != nil {
		return nil, err
	}
	out := make([]app.Activity, len(page.Records))
	for i, a := range page.Records {
		out[i] = a.toApp()
	}
	return out, nil
}

func (c *Client) Activity(ctx context.Context, accessToken string, id uuid.UUID) (app.Activity, error) {
	var a activity
	if err := c.get(ctx, accessToken, "/developer/v2/activity/workout/"+id.String(), nil, &a); err != nil {
		return app.Activity{}, err
	}
	return a.toApp(), nil
}

func (c *Client) LatestRecovery(ctx context.Context, accessToken string, from, to time.Time) (*app.Recovery, error) {
	var page struct {
		Records []struct {
			Score *struct {
				RecoveryScore    float64 `json:"recovery_score"`
				RestingHeartRate float64 `json:"resting_heart_rate"`
				HrvRmssdMilli    float64 `json:"hrv_rmssd_milli"`
			} `json:"score"`
		} `json:"records"`
	}
	query := url.Values{"start": {from.Format(time.RFC3339)}, "end": {to.Format(time.RFC3339)}, "limit": {"1"}}
	if err := c.get(ctx, accessToken, "/developer/v2/recovery", query, &page); err != nil {
		return nil, err
	}
	if len(page.Records) == 0 || page.Records[0].Score == nil {
		return nil, nil
	}
	r := page.Records[0].Score
	return &app.Recovery{
		Score: int32(math.Round(r.RecoveryScore)), RestingHR: int32(math.Round(r.RestingHeartRate)), HrvMs: r.HrvRmssdMilli,
	}, nil
}

// get calls an API path with the user's access token and decodes the answer into dst.
func (c *Client) get(ctx context.Context, accessToken, path string, query url.Values, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return c.do(req, dst)
}

// do sends req and decodes a 2xx JSON answer into dst. WHOOP rejecting the credentials is
// app.ErrRejected, an unknown activity app.ErrUnknownActivity.
func (c *Client) do(req *http.Request, dst any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("whoop %s: %w", req.URL.Path, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusBadRequest && strings.HasSuffix(req.URL.Path, "/token"):
		return app.ErrRejected
	case res.StatusCode == http.StatusNotFound:
		return app.ErrUnknownActivity
	case res.StatusCode >= 300:
		return fmt.Errorf("whoop %s: status %d", req.URL.Path, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(dst)
}
