//go:build component

package tests

import (
	"net/http"
	"strings"
	"testing"

	identityclient "github.com/erdira/homegym/backend/modules/identity/api/http/client"
)

func TestSignInAndOut(t *testing.T) {
	t.Parallel()
	u := newUserSignedOut(t)
	ctx := t.Context()

	wrong, err := u.Identity.LoginWithResponse(ctx, identityclient.LoginJSONRequestBody{Email: u.Email, Password: "wrong"})
	if err != nil || wrong.StatusCode() != http.StatusUnauthorized || wrong.JSON401.Error != "invalid email or password" {
		t.Fatalf("login with a wrong password = %v %s", err, body(wrong.Body))
	}
	if me, _ := u.Identity.MeWithResponse(ctx, &identityclient.MeParams{}); me.StatusCode() != http.StatusUnauthorized {
		t.Errorf("me before login = %d, want 401", me.StatusCode())
	}

	// Emails match regardless of case and surrounding spaces.
	login, err := u.Identity.LoginWithResponse(ctx, identityclient.LoginJSONRequestBody{
		Email: "  " + strings.ToUpper(u.Email) + " ", Password: testPassword,
	})
	if err != nil || login.StatusCode() != http.StatusOK || login.JSON200.Id != u.ID || login.JSON200.Email != u.Email {
		t.Fatalf("login = %v %s", err, body(login.Body))
	}
	// Without "remember me" the cookie is a session cookie.
	if c := login.HTTPResponse.Header.Get("Set-Cookie"); !strings.Contains(c, "HttpOnly") || strings.Contains(c, "Max-Age") {
		t.Errorf("Set-Cookie = %q, want an HttpOnly session cookie", c)
	}

	me, err := u.Identity.MeWithResponse(ctx, &identityclient.MeParams{})
	if err != nil || me.StatusCode() != http.StatusOK || me.JSON200.Id != u.ID {
		t.Fatalf("me = %v %s", err, body(me.Body))
	}

	logout, err := u.Identity.LogoutWithResponse(ctx)
	if err != nil || logout.StatusCode() != http.StatusNoContent {
		t.Fatalf("logout = %v %d", err, logout.StatusCode())
	}
	if me, _ := u.Identity.MeWithResponse(ctx, &identityclient.MeParams{}); me.StatusCode() != http.StatusUnauthorized {
		t.Errorf("me after logout = %d, want 401", me.StatusCode())
	}
}

func TestRememberMe(t *testing.T) {
	t.Parallel()
	u := newUserSignedOut(t)
	remember := true
	login, err := u.Identity.LoginWithResponse(t.Context(), identityclient.LoginJSONRequestBody{
		Email: u.Email, Password: testPassword, Remember: &remember,
	})
	if err != nil || login.StatusCode() != http.StatusOK {
		t.Fatalf("login = %v %s", err, body(login.Body))
	}
	if c := login.HTTPResponse.Header.Get("Set-Cookie"); !strings.Contains(c, "Max-Age=86") {
		t.Errorf("Set-Cookie = %q, want a cookie for the token's 24 h", c)
	}
	// Restoring the session keeps it a remembered one.
	me, _ := u.Identity.MeWithResponse(t.Context(), &identityclient.MeParams{})
	if c := me.HTTPResponse.Header.Get("Set-Cookie"); me.StatusCode() != http.StatusOK || !strings.Contains(c, "Max-Age=") {
		t.Errorf("me = %d, Set-Cookie %q", me.StatusCode(), c)
	}
}

// Every route outside identity needs a session; a bad token is rejected like none.
func TestRoutesNeedASession(t *testing.T) {
	t.Parallel()
	for _, token := range []string{"", "garbage"} {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, apiURL()+"/exercises", nil)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "token", Value: token})
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET /exercises with token %q = %d, want 401", token, resp.StatusCode)
		}
	}

	u := newUser(t)
	resp, err := u.HTTP.Get(apiURL() + "/exercises")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /exercises signed in = %d, want 200", resp.StatusCode)
	}
}
