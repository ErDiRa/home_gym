// Package module implements the whoop module's contract (client.Whoop).
package module

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/whoop/api/module/client"
	"github.com/erdira/homegym/backend/modules/whoop/app"
)

type Whoop struct {
	service *app.Service
}

var _ client.Whoop = (*Whoop)(nil)

func New(service *app.Service) *Whoop {
	if service == nil {
		panic("service cannot be nil")
	}
	return &Whoop{service: service}
}

func (w *Whoop) DailyEnergy(ctx context.Context, userID uuid.UUID, from, to time.Time, loc *time.Location) (map[string]float64, error) {
	return w.service.DailyEnergy(ctx, userID, from, to, loc)
}
