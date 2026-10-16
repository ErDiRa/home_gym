package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// AdminLookup reports whether an admin account exists. It is checked against the database
// on every request, so removing an admin takes effect immediately.
type AdminLookup func(ctx context.Context, adminID uuid.UUID) (bool, error)

// RequireAdmin allows only admin sessions through. It must run after Middleware.
func RequireAdmin(isAdmin AdminLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				unauthorized(w, "not authenticated")
				return
			}
			if claims.Role != RoleAdmin {
				writeJSONError(w, http.StatusForbidden, "admin access required")
				return
			}
			adminID, err := claims.UserID()
			if err != nil {
				unauthorized(w, "invalid token")
				return
			}
			admin, err := isAdmin(r.Context(), adminID)
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

// UserLookup reports whether a user account exists (and is not deleted). Like AdminLookup it
// is checked on every request, so deleting a user ends their sessions immediately.
type UserLookup func(ctx context.Context, userID uuid.UUID) (bool, error)

// RequireUser allows only user sessions through (admins have no user data to act on).
// It must run after Middleware.
func RequireUser(userExists UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				unauthorized(w, "not authenticated")
				return
			}
			if claims.Role != RoleUser {
				writeJSONError(w, http.StatusForbidden, "user access required")
				return
			}
			userID, err := claims.UserID()
			if err != nil {
				unauthorized(w, "invalid token")
				return
			}
			exists, err := userExists(r.Context(), userID)
			switch {
			case err != nil:
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			case !exists:
				unauthorized(w, "account no longer exists")
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
