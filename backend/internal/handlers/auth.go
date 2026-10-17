package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

// LoginAdmin signs an admin in to the admin app; user accounts fail like wrong credentials.
func (h *Handler) LoginAdmin(w http.ResponseWriter, r *http.Request) {
	h.login(w, r, h.accounts.LoginAdmin)
}

// LoginUser signs a user in to the client app; admin accounts fail like wrong credentials.
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	h.login(w, r, h.accounts.LoginUser)
}

// login checks the credentials with check and starts a new session; the login time becomes the
// token's auth_time, from which the 7-day cap is measured.
func (h *Handler) login(w http.ResponseWriter, r *http.Request, check func(ctx context.Context, email, password string) (account.Account, error)) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	acc, err := check(r.Context(), req.Email, req.Password)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	h.setSession(w, r, acc, time.Now(), req.Remember)
}

// setSession issues a fresh token and cookie for the account and writes it as the response.
// The token lasts auth.SessionTTL but never beyond auth.MaxSessionAge after authTime. With
// "remember me" the cookie expires with the token; otherwise it is dropped when the browser closes.
func (h *Handler) setSession(w http.ResponseWriter, r *http.Request, acc account.Account, authTime time.Time, remember bool) {
	now := time.Now()
	expiry := auth.SessionExpiry(now, authTime)
	token, err := h.tokens.Issue(acc.Role(), acc.ID, authTime, remember, expiry)
	if err != nil {
		serverError(w, r, err)
		return
	}
	var cookieTTL time.Duration // zero = session cookie
	if remember {
		cookieTTL = expiry.Sub(now)
	}
	auth.SetCookie(w, token, cookieTTL, h.secureCookies)
	writeJSON(w, http.StatusOK, acc)
}

// Logout clears the cookie. The JWT itself stays valid until it expires (there is no server-side
// session to revoke), but the browser no longer has it.
func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	auth.ClearCookie(w, h.secureCookies)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user and extends the session (up to auth.MaxSessionAge
// after login). The frontend uses it to restore the session since it cannot read the
// HttpOnly cookie.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id, err := claims.UserID()
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthenticated")
		return
	}
	acc, err := h.accounts.Current(r.Context(), claims.Role, id)
	// The account was deleted since login: end the session instead of extending it.
	var svcErr *service.Error
	if errors.As(err, &svcErr) && svcErr.Kind == service.KindNotFound {
		auth.ClearCookie(w, h.secureCookies)
		clientError(w, r, http.StatusUnauthorized, svcErr.Msg, svcErr.Err)
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.setSession(w, r, acc, time.Unix(claims.AuthTime, 0), claims.Remember)
}
