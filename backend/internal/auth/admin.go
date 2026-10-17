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
				unauthorized(w, r, "not authenticated", nil)
				return
			}
			if claims.Role != RoleAdmin {
				writeJSONError(w, r, http.StatusForbidden, "admin access required", nil)
				return
			}
			adminID, err := claims.UserID()
			if err != nil {
				unauthorized(w, r, "invalid token", err)
				return
			}
			admin, err := isAdmin(r.Context(), adminID)
			switch {
			case err != nil:
				writeJSONError(w, r, http.StatusInternalServerError, "internal server error", err)
			case !admin:
				writeJSONError(w, r, http.StatusForbidden, "admin access required", nil)
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
				unauthorized(w, r, "not authenticated", nil)
				return
			}
			if claims.Role != RoleUser {
				writeJSONError(w, r, http.StatusForbidden, "user access required", nil)
				return
			}
			userID, err := claims.UserID()
			if err != nil {
				unauthorized(w, r, "invalid token", err)
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
