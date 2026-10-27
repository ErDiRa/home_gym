// Package planning is the planning module: a user's cycles, trainings with their sessions and week
// plans, and the session library.
package planning

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
	planningdb "github.com/erdira/homegym/backend/modules/planning/adapters/db"
	planninghttp "github.com/erdira/homegym/backend/modules/planning/api/http"
	planningmodule "github.com/erdira/homegym/backend/modules/planning/api/module"
	"github.com/erdira/homegym/backend/modules/planning/app"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

type Module struct {
	pool      *pgxpool.Pool
	contracts *contracts.Contracts

	server *planninghttp.Server
}

// NewModule builds the module; it calls the other modules through contracts, filled in by the
// time requests arrive.
func NewModule(pool *pgxpool.Pool, contracts *contracts.Contracts) *Module {
	return &Module{pool: pool, contracts: contracts}
}

func (m *Module) Name() module.Name { return "planning" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	cat := catalogContract{m.contracts}
	repo := planningdb.NewRepo(m.pool)
	m.server = planninghttp.NewServer(app.NewService(repo, repo, repo, cat), planningdb.NewReadModel(m.pool, cat))
	return nil
}

func (m *Module) RegisterContracts(_ context.Context, c *contracts.Contracts) error {
	c.Planning = planningmodule.New(m.pool)
	return nil
}

func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error {
	planninghttp.Register(api, m.server, requireUser)
	return nil
}

// catalogContract is the catalog's contract, looked up in the registry on every call (it is set
// after this module is built).
type catalogContract struct{ c *contracts.Contracts }

func (c catalogContract) Exercises(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]catalog.Exercise, error) {
	return c.c.Catalog.Exercises(ctx, userID, ids)
}

func (c catalogContract) BlockKinds(ctx context.Context) ([]catalog.BlockKind, error) {
	return c.c.Catalog.BlockKinds(ctx)
}

func (c catalogContract) Units(ctx context.Context) (catalog.Units, error) {
	return c.c.Catalog.Units(ctx)
}
