// Package tracking is the tracking module: what a user did: workouts started from a planned slot,
// their sets and WOD scores, and the 1RM history.
package tracking

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
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
	trackingdb "github.com/erdira/homegym/backend/modules/tracking/adapters/db"
	trackinghttp "github.com/erdira/homegym/backend/modules/tracking/api/http"
	trackingmodule "github.com/erdira/homegym/backend/modules/tracking/api/module"
	"github.com/erdira/homegym/backend/modules/tracking/app"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

type Module struct {
	pool      *pgxpool.Pool
	contracts *contracts.Contracts

	server *trackinghttp.Server
}

// NewModule builds the module; it calls the other modules through contracts, filled in by the
// time requests arrive.
func NewModule(pool *pgxpool.Pool, contracts *contracts.Contracts) *Module {
	return &Module{pool: pool, contracts: contracts}
}

func (m *Module) Name() module.Name { return "tracking" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	cat, plans := catalogContract{m.contracts}, planningContract{m.contracts}
	repo := trackingdb.NewRepo(m.pool)
	m.server = trackinghttp.NewServer(app.NewService(repo, repo, plans, cat), trackingdb.NewReadModel(m.pool, cat))
	return nil
}

func (m *Module) RegisterContracts(_ context.Context, c *contracts.Contracts) error {
	c.Tracking = trackingmodule.New(m.pool)
	return nil
}

func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error {
	trackinghttp.Register(api, m.server, requireUser)
	return nil
}

// catalogContract and planningContract are the other modules' contracts, looked up in the
// registry on every call (they are set after this module is built).
type catalogContract struct{ c *contracts.Contracts }

func (c catalogContract) Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error) {
	return c.c.Catalog.Exercises(ctx, userID, ids)
}

func (c catalogContract) Units(ctx context.Context) (catalog.Units, error) {
	return c.c.Catalog.Units(ctx)
}

func (c catalogContract) ExerciseProfiles(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.ExerciseProfile, error) {
	return c.c.Catalog.ExerciseProfiles(ctx, userID, ids)
}

type planningContract struct{ c *contracts.Contracts }

func (p planningContract) ActiveTrainings(ctx context.Context, userID uuid.UUID) ([]planning.ActiveTraining, error) {
	return p.c.Planning.ActiveTrainings(ctx, userID)
}

func (p planningContract) WeekPlan(ctx context.Context, userID, trainingID uuid.UUID, week int32) ([]planning.PlannedSlot, error) {
	return p.c.Planning.WeekPlan(ctx, userID, trainingID, week)
}

func (p planningContract) Slot(ctx context.Context, userID, slotID uuid.UUID) (planning.Slot, error) {
	return p.c.Planning.Slot(ctx, userID, slotID)
}

func (p planningContract) Cycle(ctx context.Context, userID, cycleID uuid.UUID) (planning.Cycle, error) {
	return p.c.Planning.Cycle(ctx, userID, cycleID)
}
