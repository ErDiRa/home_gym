package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMiddleware(t *testing.T) {
	tm, err := NewTokenManager("test-secret-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	expired, _ := NewTokenManager("test-secret-test-secret")
	other, _ := NewTokenManager("another-secret-another")

	userID := uuid.New()
	valid, _ := tm.Issue(RoleUser, userID, time.Now(), false, time.Now().Add(time.Hour))
	expiredTok, _ := expired.Issue(RoleUser, userID, time.Now(), false, time.Now().Add(-time.Hour))
	foreign, _ := other.Issue(RoleUser, userID, time.Now(), false, time.Now().Add(time.Hour))

	tests := []struct {
		name   string
		cookie string
		want   int
	}{
		{"valid token", valid, http.StatusOK},
		{"missing cookie", "", http.StatusUnauthorized},
		{"expired token", expiredTok, http.StatusUnauthorized},
		{"wrong signature", foreign, http.StatusUnauthorized},
		{"garbage", "nope", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Middleware(tm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if id, ok := UserIDFromContext(r.Context()); !ok || id != userID {
					t.Errorf("user id not in context: %v %v", id, ok)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CookieName, Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestSessionExpiry(t *testing.T) {
	login := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Shortly after login the session is extended by the full TTL.
	now := login.Add(time.Hour)
	if got, want := SessionExpiry(now, login), now.Add(SessionTTL); !got.Equal(want) {
		t.Errorf("early refresh = %v, want %v", got, want)
	}

	// Near the cap the extension is cut off at login + MaxSessionAge.
	now = login.Add(MaxSessionAge - time.Hour)
	if got, want := SessionExpiry(now, login), login.Add(MaxSessionAge); !got.Equal(want) {
		t.Errorf("capped refresh = %v, want %v", got, want)
	}
}

func TestRequireAdmin(t *testing.T) {
	tm, _ := NewTokenManager("test-secret-test-secret")
	adminToken, _ := tm.Issue(RoleAdmin, uuid.New(), time.Now(), false, time.Now().Add(time.Hour))
	userToken, _ := tm.Issue(RoleUser, uuid.New(), time.Now(), false, time.Now().Add(time.Hour))

	tests := []struct {
		name   string
		token  string
		lookup AdminLookup
		want   int
	}{
		{"admin", adminToken, func(context.Context, uuid.UUID) (bool, error) { return true, nil }, http.StatusOK},
		{"admin removed", adminToken, func(context.Context, uuid.UUID) (bool, error) { return false, nil }, http.StatusForbidden},
		{"lookup failure", adminToken, func(context.Context, uuid.UUID) (bool, error) { return false, errors.New("db down") }, http.StatusInternalServerError},
		{"user token", userToken, func(context.Context, uuid.UUID) (bool, error) { return true, nil }, http.StatusForbidden},
		{"no cookie", "", func(context.Context, uuid.UUID) (bool, error) { return true, nil }, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Middleware(tm)(RequireAdmin(tt.lookup)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{Name: CookieName, Value: tt.token})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
