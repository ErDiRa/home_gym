// Package module implements the identity module's contract (client.Identity).
package module

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/modules/identity/api/module/client"
	"github.com/erdira/homegym/backend/modules/identity/app"
)

type Identity struct {
	service *app.Service
}

var _ client.Identity = (*Identity)(nil)

func New(service *app.Service) *Identity {
	if service == nil {
		panic("service cannot be nil")
	}
	return &Identity{service: service}
}

func (i *Identity) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	return i.service.Authenticate(ctx, token)
}
