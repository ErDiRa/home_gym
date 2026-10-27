package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/logging"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/account"
)

// Login checks the credentials and starts a new session; the login time becomes the token's
// auth_time, from which the 7-day cap is measured.
func (h *Handler) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	acc, err := h.accounts.Login(ctx, req.Body.Email, req.Body.Password)
	if err != nil {
		return nil, err
	}
	cookie, err := h.sessionCookie(acc.ID, time.Now(), value(req.Body.Remember))
	if err != nil {
		return nil, err
	}
	return api.Login200JSONResponse{Body: accountToAPI(acc), Headers: api.Login200ResponseHeaders{SetCookie: &cookie}}, nil
}

// sessionCookie issues a fresh token and returns its cookie (as a Set-Cookie value). The token
// lasts auth.SessionTTL but never beyond auth.MaxSessionAge after authTime. With "remember me"
// the cookie expires with the token; otherwise it is dropped when the browser closes.
func (h *Handler) sessionCookie(userID uuid.UUID, authTime time.Time, remember bool) (string, error) {
	now := time.Now()
	expiry := auth.SessionExpiry(now, authTime)
	token, err := h.tokens.Issue(userID, authTime, remember, expiry)
	if err != nil {
		return "", err
	}
	var cookieTTL time.Duration // zero = session cookie
	if remember {
		cookieTTL = expiry.Sub(now)
	}
	return auth.Cookie(token, cookieTTL, h.secureCookies).String(), nil
}

// Logout clears the cookie. The JWT itself stays valid until it expires (there is no server-side
// session to revoke), but the browser no longer has it.
func (h *Handler) Logout(context.Context, api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	cleared := auth.ClearedCookie(h.secureCookies).String()
	return api.Logout204Response{Headers: api.Logout204ResponseHeaders{SetCookie: &cleared}}, nil
}

// Me returns the authenticated user and extends the session (up to auth.MaxSessionAge
// after login). The frontend uses it to restore the session since it cannot read the
// HttpOnly cookie.
func (h *Handler) Me(ctx context.Context, _ api.MeRequestObject) (api.MeResponseObject, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, service.Unauthorized("unauthenticated")
	}
	id, err := claims.UserID()
	if err != nil {
		return nil, service.Unauthorized("unauthenticated")
	}
	acc, err := h.accounts.Current(ctx, id)
	// The account was deleted since login: end the session instead of extending it.
	var svcErr *service.Error
	if errors.As(err, &svcErr) && svcErr.Kind == service.KindNotFound {
		logging.ContextError(ctx, http.StatusUnauthorized, svcErr.Msg, svcErr.Err)
		cleared := auth.ClearedCookie(h.secureCookies).String()
		return api.Me401JSONResponse{Body: api.Error{Error: svcErr.Msg}, Headers: api.Me401ResponseHeaders{SetCookie: &cleared}}, nil
	}
	if err != nil {
		return nil, err
	}
	cookie, err := h.sessionCookie(acc.ID, time.Unix(claims.AuthTime, 0), claims.Remember)
	if err != nil {
		return nil, err
	}
	return api.Me200JSONResponse{Body: accountToAPI(acc), Headers: api.Me200ResponseHeaders{SetCookie: &cookie}}, nil
}

func accountToAPI(a account.Account) api.Account {
	return api.Account{Id: a.ID, Email: a.Email}
}
