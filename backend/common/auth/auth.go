// Package auth carries the signed-in user through a request: the cookie the session token travels
// in and the user's id in the context. Issuing and checking tokens is the identity module's.
package auth

import (
	"context"

	"github.com/google/uuid"
)

// CookieName is the HttpOnly cookie that carries the session token.
const CookieName = "token"

type ctxKey struct{}

// WithUserID returns ctx carrying the signed-in user's id (set by web.RequireUser; tests set it
// directly).
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserIDFromContext returns the signed-in user's id, if the request has one.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	return id, ok
}
