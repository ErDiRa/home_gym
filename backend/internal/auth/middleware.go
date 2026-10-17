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
// (account ID and role) in the request context. Roles are checked by RequireUser/RequireAdmin.
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
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
		})
	}
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

// SetCookie stores the token in an HttpOnly cookie. A zero ttl makes it a
// session cookie that the browser drops on exit.
func SetCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
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
	http.SetCookie(w, c)
}

// ClearCookie deletes the auth cookie. Its attributes must match SetCookie's, or the browser
// treats it as a different cookie and keeps the old one.
func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
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
