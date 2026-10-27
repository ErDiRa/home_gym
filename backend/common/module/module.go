// Package module defines what a module of the monolith provides to the service that wires them
// (backend.New).
package module

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/common/module/contracts"
)

// Name is a module's name, which is also its Postgres schema.
type Name string

// Module is one part of the monolith: its own schema, business logic and HTTP routes. The
// service calls Init on every module (in order: a module's migrations may reference the tables of
// the modules before it), then RegisterContracts on every module, then RegisterHTTP; so a module
// may call other modules' contracts from RegisterHTTP on, never during Init.
type Module interface {
	Name() Name
	// Init applies the module's migrations and builds its services.
	Init(ctx context.Context) error
	// RegisterContracts sets the module's contract (what other modules may call) in c.
	RegisterContracts(ctx context.Context, c *contracts.Contracts) error
	// RegisterHTTP mounts the module's routes on api, the router of /api. requireUser is the
	// middleware that lets only signed-in users through, with their id in the context.
	RegisterHTTP(ctx context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error
}
