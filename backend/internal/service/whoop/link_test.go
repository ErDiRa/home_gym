package whoop

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

const scoredWorkout = `{"id":"ecfc6a15-4661-442f-a9a4-f160dd7afae8","start":"2026-10-20T17:00:00Z","end":"2026-10-20T18:05:00Z",
"sport_name":"weightlifting","score_state":"SCORED","score":{"strain":11.4,"average_heart_rate":121,"max_heart_rate":172,
"kilojoule":1850.5,"zone_durations":{"zone_zero_milli":60000,"zone_one_milli":600000,"zone_two_milli":1200000,
"zone_three_milli":900000,"zone_four_milli":300000,"zone_five_milli":0}}}`

func TestLink(t *testing.T) {
	workoutID := uuid.New()
	whoopID := uuid.MustParse("ecfc6a15-4661-442f-a9a4-f160dd7afae8")
	f := &fakeQueries{
		workouts: map[uuid.UUID]bool{workoutID: true},
		conn:     &repos.WhoopConnection{AccessToken: "valid", ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}},
	}
	var recoveryStart string
	s := newTestService(t, f, map[string]http.HandlerFunc{
		"/developer/v2/activity/workout/" + whoopID.String(): func(w http.ResponseWriter, r *http.Request) {
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

	if _, err := s.Link(context.Background(), uuid.New(), workoutID, whoopID); err != nil {
		t.Fatal(err)
	}
	if len(f.saved) != 1 {
		t.Fatalf("saved %d rows, want 1", len(f.saved))
	}
	got := f.saved[0]
	if got.WhoopWorkoutID != whoopID || got.SportName != "weightlifting" || *got.Strain != 11.4 || *got.AvgHr != 121 ||
		*got.MaxHr != 172 || *got.Kilojoule != 1850.5 || got.ZoneMs[2] != 1200000 || len(got.ZoneMs) != 6 {
		t.Errorf("saved = %+v", got)
	}
	if *got.RecoveryScore != 68 || *got.RestingHr != 52 || *got.HrvMs != 61.3 {
		t.Errorf("recovery = %v %v %v", *got.RecoveryScore, *got.RestingHr, *got.HrvMs)
	}
	if recoveryStart != "2026-10-19T17:00:00Z" {
		t.Errorf("recovery from %s, want the day before the activity", recoveryStart)
	}

	// Another user's workout looks missing.
	_, err := s.Link(context.Background(), uuid.New(), uuid.New(), whoopID)
	wantKind(t, err, service.KindNotFound)
}

func TestLinkPending(t *testing.T) {
	workoutID := uuid.New()
	f := &fakeQueries{
		workouts: map[uuid.UUID]bool{workoutID: true},
		conn:     &repos.WhoopConnection{AccessToken: "valid", ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}},
	}
	whoopID := uuid.New()
	s := newTestService(t, f, map[string]http.HandlerFunc{
		"/developer/v2/activity/workout/" + whoopID.String(): func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(`{"id":"` + whoopID.String() + `","start":"2026-10-20T17:00:00Z","end":"2026-10-20T18:00:00Z","sport_name":"functional-fitness","score_state":"PENDING_SCORE"}`))
		},
		"/developer/v2/recovery": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"records":[]}`)) },
	})
	if _, err := s.Link(context.Background(), uuid.New(), workoutID, whoopID); err != nil {
		t.Fatal(err)
	}
	if got := f.saved[0]; got.ScoreState != "PENDING_SCORE" || got.Strain != nil || got.ZoneMs != nil || got.RecoveryScore != nil {
		t.Errorf("saved = %+v", got)
	}
}

func TestAccessTokenRefresh(t *testing.T) {
	f := &fakeQueries{conn: &repos.WhoopConnection{AccessToken: "old", RefreshToken: "refresh-1", ExpiresAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}}
	var form string
	s := newTestService(t, f, map[string]http.HandlerFunc{
		"/oauth/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			form = r.Form.Get("grant_type") + " " + r.Form.Get("refresh_token") + " " + r.Form.Get("client_id")
			w.Write([]byte(`{"access_token":"new","refresh_token":"refresh-2","expires_in":3600}`))
		},
	})
	token, err := s.accessToken(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if token != "new" || form != "refresh_token refresh-1 id" {
		t.Errorf("token %q after refreshing with %q", token, form)
	}
	if f.conn.RefreshToken != "refresh-2" || time.Until(f.conn.ExpiresAt.Time) < 59*time.Minute {
		t.Errorf("stored %+v, want the rotated refresh token", f.conn)
	}

	// A refresh token WHOOP no longer accepts asks to reconnect.
	f.conn.ExpiresAt.Time = time.Now()
	s = newTestService(t, f, map[string]http.HandlerFunc{
		"/oauth/oauth2/token": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
	})
	_, err = s.accessToken(context.Background(), uuid.New())
	wantKind(t, err, service.KindUnauthorized)

	// Without a connection there is nothing to call WHOOP with.
	f.conn = nil
	_, err = s.accessToken(context.Background(), uuid.New())
	wantKind(t, err, service.KindInvalid)
}

func wantKind(t *testing.T, err error, kind service.Kind) {
	t.Helper()
	var svcErr *service.Error
	if !errors.As(err, &svcErr) || svcErr.Kind != kind {
		t.Errorf("err = %v, want a service error of kind %d", err, kind)
	}
}
