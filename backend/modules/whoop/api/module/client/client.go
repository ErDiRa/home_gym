// Package client is the whoop module's contract: what other modules may read of the WHOOP data.
package client

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Whoop interface {
	// DailyEnergy returns the energy (kJ) WHOOP measured on the user's local days from from (a
	// day's start in loc) up to to, keyed by the day (YYYY-MM-DD). Days WHOOP hasn't finished and
	// scored are missing, every day without a connection.
	DailyEnergy(ctx context.Context, userID uuid.UUID, from, to time.Time, loc *time.Location) (map[string]float64, error)
}
