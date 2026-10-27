// Package catalog is the catalog module: each user's exercises with their ratings, and the shared
// lists trainings are planned with (joints, muscle groups, focuses, block kinds, units).
package catalog

import (
	"context"
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/db"
	"github.com/erdira/homegym/backend/common/module"
	"github.com/erdira/homegym/backend/common/module/contracts"
	catalogdb "github.com/erdira/homegym/backend/modules/catalog/adapters/db"
	cataloghttp "github.com/erdira/homegym/backend/modules/catalog/api/http"
	catalogmodule "github.com/erdira/homegym/backend/modules/catalog/api/module"
	"github.com/erdira/homegym/backend/modules/catalog/app"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

type Module struct {
	pool *pgxpool.Pool

	server *cataloghttp.Server
}

func NewModule(pool *pgxpool.Pool) *Module {
	return &Module{pool: pool}
}

func (m *Module) Name() module.Name { return "catalog" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	repo := catalogdb.NewRepo(m.pool)
	m.server = cataloghttp.NewServer(app.NewService(repo, repo, repo), catalogdb.NewReadModel(m.pool))
	return nil
}

func (m *Module) RegisterContracts(_ context.Context, c *contracts.Contracts) error {
	c.Catalog = catalogmodule.New(m.pool)
	return nil
}

func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, requireUser func(http.Handler) http.Handler) error {
	cataloghttp.Register(api, m.server, requireUser)
	return nil
}
