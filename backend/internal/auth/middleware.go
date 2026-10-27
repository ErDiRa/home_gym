package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/logging"
)

// CookieName is the HttpOnly cookie that carries the JWT.
const CookieName = "token"

// ctxKey is unexported so no other package can read or overwrite the claims in the context.
type ctxKey struct{}

// Middleware rejects requests without a valid JWT cookie and stores the verified claims
// (the user ID) in the request context. Whether the user still exists is checked by RequireUser.
func Middleware(tokens *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				unauthorized(w, r, "not authenticated", err)
				return
			}
			claims, err := tokens.Parse(cookie.Value)
			if err != nil {
				unauthorized(w, r, "invalid or expired token", err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

// WithClaims returns ctx carrying claims, as Middleware stores them (also for handler tests).
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, claims)
}

// ClaimsFromContext returns the verified token claims stored by Middleware.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ctxKey{}).(*Claims)
	return claims, ok
}

// UserIDFromContext returns the authenticated user ID stored by Middleware.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return uuid.Nil, false
	}
	id, err := claims.UserID()
	return id, err == nil
}

// Cookie is the HttpOnly cookie storing the token. A zero ttl makes it a session cookie that
// the browser drops on exit.
func Cookie(token string, ttl time.Duration, secure bool) *http.Cookie {
	c := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	if ttl > 0 {
		c.MaxAge = int(ttl.Seconds())
	}
	return c
}

// ClearedCookie deletes the auth cookie. Its attributes must match Cookie's, or the browser
// treats it as a different cookie and keeps the old one.
func ClearedCookie(secure bool) *http.Cookie {
	c := Cookie("", 0, secure)
	c.MaxAge = -1
	return c
}

func unauthorized(w http.ResponseWriter, r *http.Request, msg string, err error) {
	writeJSONError(w, r, http.StatusUnauthorized, msg, err)
}

// writeJSONError sends a JSON error response and logs it with the underlying cause (may be nil).
func writeJSONError(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	logging.RequestError(r, status, msg, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
