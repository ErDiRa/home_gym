package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account"
)

// fakeAccounts implements AccountService for testUser with password "test".
type fakeAccounts struct{}

func (fakeAccounts) Login(_ context.Context, email, password string) (account.Account, error) {
	if password != "test" {
		return account.Account{}, service.Unauthorized("invalid email or password")
	}
	return account.Account{ID: testUser, Email: email}, nil
}

func (fakeAccounts) Current(_ context.Context, id uuid.UUID) (account.Account, error) {
	return account.Account{ID: id, Email: "me@example.com"}, nil
}

func (fakeAccounts) UserExists(context.Context, uuid.UUID) (bool, error) { return true, nil }

// Only login and logout (security: [] in the spec) are served without a session; the real
// auth middlewares guard everything else.
func TestRoutesRequireUser(t *testing.T) {
	tokens, err := auth.NewTokenManager("a test secret of enough length")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	New(Services{Accounts: fakeAccounts{}, Trainings: &fakeTrainings{}}, tokens, false).Routes(r)
	do := func(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	for _, path := range []string{"/api/auth/me", "/api/trainings/" + uuid.NewString(), "/api/whoop"} {
		if rec := do(http.MethodGet, path, ""); rec.Code != http.StatusUnauthorized || errorBody(t, rec) != "not authenticated" {
			t.Errorf("GET %s without a session = %d", path, rec.Code)
		}
	}
	if rec := do(http.MethodPost, "/api/auth/login", `{"email":"me@example.com","password":"wrong"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("login with a wrong password = %d", rec.Code)
	}
	if rec := do(http.MethodPost, "/api/auth/logout", ""); rec.Code != http.StatusNoContent || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("logout = %d, Set-Cookie %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}

	rec := do(http.MethodPost, "/api/auth/login", `{"email":"me@example.com","password":"test","remember":true}`)
	cookies := rec.Result().Cookies()
	if rec.Code != http.StatusOK || len(cookies) != 1 || cookies[0].Name != auth.CookieName || cookies[0].MaxAge <= 0 {
		t.Fatalf("login = %d %s, cookies %v", rec.Code, rec.Body, cookies)
	}
	if rec := do(http.MethodGet, "/api/trainings/"+uuid.NewString(), "", cookies[0]); rec.Code != http.StatusOK {
		t.Errorf("GET with the session = %d %s", rec.Code, rec.Body)
	}
	rec = do(http.MethodGet, "/api/auth/me", "", cookies[0])
	if renewed := rec.Result().Cookies(); rec.Code != http.StatusOK || len(renewed) != 1 || renewed[0].MaxAge > int((7*24*time.Hour).Seconds()) {
		t.Errorf("me = %d, cookies %v", rec.Code, renewed)
	}
}
