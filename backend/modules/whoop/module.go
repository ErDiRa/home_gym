// Package whoop is the whoop module: the connection to a user's WHOOP account and the WHOOP
// activity linked to a workout.
package whoop

import (
	"context"
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/db"
	"github.com/erdira/homegym/backend/common/module"
	"github.com/erdira/homegym/backend/common/module/contracts"
	whoopdb "github.com/erdira/homegym/backend/modules/whoop/adapters/db"
	"github.com/erdira/homegym/backend/modules/whoop/adapters/whoopapi"
	whoophttp "github.com/erdira/homegym/backend/modules/whoop/api/http"
	"github.com/erdira/homegym/backend/modules/whoop/app"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

// Config is what the module needs from the environment.
type Config struct {
	API           whoopapi.Config // the WHOOP app; WHOOP is off while it is incomplete
	SecureCookies bool            // mark the OAuth state cookie Secure (HTTPS only)
}

type Module struct {
	pool      *pgxpool.Pool
	contracts *contracts.Contracts
	cfg       Config

	server *whoophttp.Server
}

// NewModule builds the module; it calls the other modules through contracts, filled in by the
// time requests arrive.
func NewModule(pool *pgxpool.Pool, contracts *contracts.Contracts, cfg Config) *Module {
	return &Module{pool: pool, contracts: contracts, cfg: cfg}
}

func (m *Module) Name() module.Name { return "whoop" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	repo := whoopdb.NewRepo(m.pool)
	service := app.NewService(whoopapi.NewClient(m.cfg.API), repo, repo, trackingContract{m.contracts})
	m.server = whoophttp.NewServer(service, m.cfg.SecureCookies)
	return nil
}

func (m *Module) RegisterContracts(context.Context, *contracts.Contracts) error { return nil }

func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error {
	whoophttp.Register(api, m.server, requireUser)
	return nil
}

// trackingContract is the tracking module's contract, looked up in the registry on every call (it
// is set after this module is built).
type trackingContract struct{ c *contracts.Contracts }

func (t trackingContract) UserOwnsWorkout(ctx context.Context, userID, workoutID uuid.UUID) (bool, error) {
	return t.c.Tracking.UserOwnsWorkout(ctx, userID, workoutID)
}
