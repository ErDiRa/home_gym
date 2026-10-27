package whoopapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/whoop/app"
)

const scoredWorkout = `{"id":"ecfc6a15-4661-442f-a9a4-f160dd7afae8","start":"2026-10-20T17:00:00Z","end":"2026-10-20T18:05:00Z",
"sport_name":"weightlifting","score_state":"SCORED","score":{"strain":11.4,"average_heart_rate":121,"max_heart_rate":172,
"kilojoule":1850.5,"zone_durations":{"zone_zero_milli":60000,"zone_one_milli":600000,"zone_two_milli":1200000,
"zone_three_milli":900000,"zone_four_milli":300000,"zone_five_milli":0}}}`

// newTestClient runs the client against a fake WHOOP API answering with the handlers in routes
// (by path).
func newTestClient(t *testing.T, routes map[string]http.HandlerFunc) *Client {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range routes {
		mux.HandleFunc(path, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{cfg: Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost/cb"}, http: srv.Client(), baseURL: srv.URL}
}

func TestActivityAndRecovery(t *testing.T) {
	id := uuid.MustParse("ecfc6a15-4661-442f-a9a4-f160dd7afae8")
	var recoveryStart string
	c := newTestClient(t, map[string]http.HandlerFunc{
		"/developer/v2/activity/workout/" + id.String(): func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer valid" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(scoredWorkout))
		},
		"/developer/v2/recovery": func(w http.ResponseWriter, r *http.Request) {
			recoveryStart = r.URL.Query().Get("start")
			w.Write([]byte(`{"records":[{"score":{"recovery_score":67.6,"resting_heart_rate":52,"hrv_rmssd_milli":61.3}}]}`))
		},
	})

	a, err := c.Activity(context.Background(), "valid", id)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != id || a.SportName != "weightlifting" || *a.Strain != 11.4 || *a.AvgHR != 121 || *a.MaxHR != 172 ||
		*a.Kilojoule != 1850.5 || len(a.ZoneMs) != 6 || a.ZoneMs[2] != 1200000 {
		t.Errorf("activity = %+v", a)
	}
	r, err := c.LatestRecovery(context.Background(), "valid", a.Start.Add(-24*time.Hour), a.Start)
	if err != nil || r.Score != 68 || r.RestingHR != 52 || r.HrvMs != 61.3 {
		t.Errorf("recovery = %+v, %v", r, err)
	}
	if recoveryStart != "2026-10-19T17:00:00Z" {
		t.Errorf("recovery from %s", recoveryStart)
	}

	if _, err := c.Activity(context.Background(), "expired", id); !errors.Is(err, app.ErrRejected) {
		t.Errorf("a rejected token: %v", err)
	}
	if _, err := c.Activity(context.Background(), "valid", uuid.New()); !errors.Is(err, app.ErrUnknownActivity) {
		t.Errorf("an unknown activity: %v", err)
	}
}

func TestPendingActivity(t *testing.T) {
	id := uuid.New()
	c := newTestClient(t, map[string]http.HandlerFunc{
		"/developer/v2/activity/workout/" + id.String(): func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(`{"id":"` + id.String() + `","start":"2026-10-20T17:00:00Z","end":"2026-10-20T18:00:00Z","sport_name":"functional-fitness","score_state":"PENDING_SCORE"}`))
		},
		"/developer/v2/recovery": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"records":[]}`)) },
	})
	a, err := c.Activity(context.Background(), "valid", id)
	if err != nil || a.ScoreState != "PENDING_SCORE" || a.Strain != nil || a.ZoneMs != nil {
		t.Errorf("activity = %+v, %v", a, err)
	}
	if r, err := c.LatestRecovery(context.Background(), "valid", a.Start.Add(-24*time.Hour), a.Start); r != nil || err != nil {
		t.Errorf("recovery = %+v, %v; want none", r, err)
	}
}

func TestRefresh(t *testing.T) {
	var form string
	c := newTestClient(t, map[string]http.HandlerFunc{
		"/oauth/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			form = r.Form.Get("grant_type") + " " + r.Form.Get("refresh_token") + " " + r.Form.Get("client_id")
			if r.Form.Get("refresh_token") == "revoked" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Write([]byte(`{"access_token":"new","refresh_token":"refresh-2","expires_in":3600}`))
		},
	})
	tokens, err := c.Refresh(context.Background(), "refresh-1")
	if err != nil || tokens.AccessToken != "new" || tokens.RefreshToken != "refresh-2" || time.Until(tokens.ExpiresAt) < 59*time.Minute {
		t.Errorf("tokens = %+v, %v", tokens, err)
	}
	if form != "refresh_token refresh-1 id" {
		t.Errorf("form = %q", form)
	}
	// A refresh token WHOOP no longer accepts.
	if _, err := c.Refresh(context.Background(), "revoked"); !errors.Is(err, app.ErrRejected) {
		t.Errorf("a revoked refresh token: %v", err)
	}
}
