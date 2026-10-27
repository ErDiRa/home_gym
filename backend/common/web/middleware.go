package web

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/auth"
)

// MaxBody is the default limit of a request body.
const MaxBody = 1 << 20

// LimitBody caps request bodies at limit bytes; a larger one fails decoding (400).
func LimitBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// Authenticate checks a session token and returns its user's id; an invalid token or a user who
// no longer exists is an errs.Unauthorized error.
type Authenticate func(ctx context.Context, token string) (uuid.UUID, error)

// RequireUser lets only requests with a valid session through, with the user's id in the context
// (auth.UserIDFromContext): the token comes from the auth.CookieName cookie and is checked by
// authenticate on every request, so deleting a user ends their sessions at once. Otherwise it
// answers 401 with authenticate's message.
func RequireUser(authenticate Authenticate) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err != nil {
				ClientError(w, r, http.StatusUnauthorized, "not authenticated", err)
				return
			}
			userID, err := authenticate(r.Context(), cookie.Value)
			if err != nil {
				ServiceError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), userID)))
		})
	}
}
