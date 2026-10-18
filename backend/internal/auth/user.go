package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// UserLookup reports whether a user account exists (and is not deleted). It is checked on
// every request, so deleting a user ends their sessions immediately.
type UserLookup func(ctx context.Context, userID uuid.UUID) (bool, error)

// RequireUser allows only sessions of existing users through. It must run after Middleware.
func RequireUser(userExists UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				unauthorized(w, r, "not authenticated", nil)
				return
			}
			exists, err := userExists(r.Context(), userID)
			switch {
			case err != nil:
				writeJSONError(w, r, http.StatusInternalServerError, "internal server error", err)
			case !exists:
				unauthorized(w, r, "account no longer exists", nil)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
