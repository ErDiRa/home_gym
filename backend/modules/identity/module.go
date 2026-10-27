// Package identity is the identity module: the users who sign in, and their sessions (login, JWT
// cookie).
package identity

import (
	"context"
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/db"
	"github.com/erdira/homegym/backend/common/module"
	"github.com/erdira/homegym/backend/common/module/contracts"
	identitydb "github.com/erdira/homegym/backend/modules/identity/adapters/db"
	"github.com/erdira/homegym/backend/modules/identity/adapters/jwt"
	identityhttp "github.com/erdira/homegym/backend/modules/identity/api/http"
	identitymodule "github.com/erdira/homegym/backend/modules/identity/api/module"
	"github.com/erdira/homegym/backend/modules/identity/app"
)

//go:embed adapters/db/migrations/*.sql
var migrations embed.FS

// Config is what the module needs from the environment.
type Config struct {
	JWTSecret     string // signs the session tokens; at least 16 characters
	SecureCookies bool   // mark the session cookie Secure (HTTPS only)
}

type Module struct {
	pool *pgxpool.Pool
	cfg  Config

	service *app.Service
	server  *identityhttp.Server
}

func NewModule(pool *pgxpool.Pool, cfg Config) *Module {
	return &Module{pool: pool, cfg: cfg}
}

func (m *Module) Name() module.Name { return "identity" }

func (m *Module) Init(ctx context.Context) error {
	if err := db.MigrateUp(ctx, string(m.Name()), m.pool, migrations, "adapters/db/migrations"); err != nil {
		return err
	}
	tokens, err := jwt.NewTokens(m.cfg.JWTSecret)
	if err != nil {
		return err
	}
	m.service = app.NewService(identitydb.NewUsersRepo(m.pool), tokens)
	m.server = identityhttp.NewServer(m.service, m.cfg.SecureCookies)
	return nil
}

func (m *Module) RegisterContracts(_ context.Context, c *contracts.Contracts) error {
	c.Identity = identitymodule.New(m.service)
	return nil
}

// RegisterHTTP mounts login, logout and me; they don't need requireUser (me checks the cookie
// itself).
func (m *Module) RegisterHTTP(_ context.Context, api chi.Router, _ func(http.Handler) http.Handler) error {
	identityhttp.Register(api, m.server)
	return nil
}
