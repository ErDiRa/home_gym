package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// AdminLookup reports whether a user is an admin. It is checked against the database on
// every request, so revoking admin rights takes effect immediately.
type AdminLookup func(ctx context.Context, userID uuid.UUID) (bool, error)

// RequireAdmin allows only admins through. It must run after Middleware.
func RequireAdmin(isAdmin AdminLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				unauthorized(w, "not authenticated")
				return
			}
			admin, err := isAdmin(r.Context(), userID)
			switch {
			case err != nil:
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			case !admin:
				writeJSONError(w, http.StatusForbidden, "admin access required")
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
