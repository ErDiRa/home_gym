// Package client is the identity module's contract: what other modules (and the service wiring)
// may call.
package client

import (
	"context"

	"github.com/google/uuid"
)

type Identity interface {
	// Authenticate checks a session token and returns its user's id. An invalid or expired token,
	// or a user who no longer exists, is an errs.Unauthorized error.
	Authenticate(ctx context.Context, token string) (uuid.UUID, error)
}
