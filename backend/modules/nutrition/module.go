// Package nutrition is the nutrition module: a user's body (profile and weight history), the
// macros they ate per day and the daily targets estimated from them.
package nutrition

import (
	"context"
	"embed"
	"net/http"
	"time"
	_ "time/tzdata" // the clients' time zones, also in an image without them

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/db"
	"github.com/erdira/homegym/backend/common/module"
	"github.com/erdira/homegym/backend/common/module/contracts"
	nutritiondb "github.com/erdira/homegym/backend/modules/nutrition/adapters/db"
	nutritionhttp "github.com/erdira/homegym/backend/modules/nutrition/api/http"
	"github.com/erdira/homegym/backend/modules/nutrition/app"
	tracking "github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

type Module struct {
	pool      *pgxpool.Pool
	contracts *contracts.Contracts

	server *nutritionhttp.Server
}

// NewModule builds the module; it calls the other modules through contracts, filled in by the
// time requests arrive.
func NewModule(pool *pgxpool.Pool, contracts *contracts.Contracts) *Module {
	return &Module{pool: pool, contracts: contracts}
}

func (m *Module) Name() module.Name { return "nutrition" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	service := app.NewService(nutritiondb.NewRepo(m.pool), trackingContract{m.contracts}, whoopContract{m.contracts})
	m.server = nutritionhttp.NewServer(service, nutritiondb.NewReadModel(m.pool))
	return nil
}

func (m *Module) RegisterContracts(context.Context, *contracts.Contracts) error { return nil }

func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error {
	nutritionhttp.Register(api, m.server, requireUser)
	return nil
}

// trackingContract and whoopContract are the other modules' contracts, looked up in the registry
// on every call.
type trackingContract struct{ c *contracts.Contracts }

func (t trackingContract) FinishedWorkouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]tracking.FinishedWorkout, error) {
	return t.c.Tracking.FinishedWorkouts(ctx, userID, from, to)
}

type whoopContract struct{ c *contracts.Contracts }

func (w whoopContract) DailyEnergy(ctx context.Context, userID uuid.UUID, from, to time.Time, loc *time.Location) (map[string]float64, error) {
	return w.c.Whoop.DailyEnergy(ctx, userID, from, to, loc)
}
