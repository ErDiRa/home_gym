// Package backend wires the modules into one service: one database (a schema per module), one
// HTTP server (/api, plus the built frontend in production).
package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/common/log"
	"github.com/erdira/homegym/backend/common/module"
	"github.com/erdira/homegym/backend/common/module/contracts"
	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/catalog"
	"github.com/erdira/homegym/backend/modules/identity"
	"github.com/erdira/homegym/backend/modules/planning"
	"github.com/erdira/homegym/backend/modules/tracking"
	"github.com/erdira/homegym/backend/modules/whoop"
	"github.com/erdira/homegym/backend/modules/whoop/adapters/whoopapi"
)

// Config is what the service needs from the environment.
type Config struct {
	JWTSecret    string
	CookieSecure bool            // mark cookies Secure (HTTPS only)
	CORSOrigins  []string        // origins allowed to call the API with the cookie
	StaticDir    string          // the built frontend to serve next to the API; empty in dev (Vite serves it)
	Whoop        whoopapi.Config // the WHOOP app; WHOOP is off while it is incomplete
}

// Svc is the wired service; Run serves it.
type Svc struct {
	router http.Handler
	logger *slog.Logger
}

// NewPool connects to the database. The modules' queries name their schema, so the search path is
// left as it is.
func NewPool(ctx context.Context, dbSource string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dbSource)
	if err != nil {
		return nil, err
	}
	// pgxpool connects lazily; ping so a bad DB_SOURCE fails at startup, not on the first request.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// New migrates the database and wires the modules: Init (each module's migrations, in order),
// RegisterContracts, then RegisterHTTP.
func New(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, cfg Config) (*Svc, error) {
	if err := legacy.Upgrade(ctx, pool); err != nil {
		return nil, err
	}

	// Filled in by the modules (RegisterContracts); they call each other through it afterwards.
	moduleContracts := &contracts.Contracts{}
	// Migrations run in this order: a module's tables may reference the ones before it.
	modules := []module.Module{
		identity.NewModule(pool, identity.Config{JWTSecret: cfg.JWTSecret, SecureCookies: cfg.CookieSecure}),
		catalog.NewModule(pool),
		planning.NewModule(pool, moduleContracts),
		tracking.NewModule(pool, moduleContracts),
		whoop.NewModule(pool, moduleContracts, whoop.Config{API: cfg.Whoop, SecureCookies: cfg.CookieSecure}),
	}
	for _, m := range modules {
		start := time.Now()
		if err := m.Init(ctx); err != nil {
			return nil, fmt.Errorf("init module %s: %w", m.Name(), err)
		}
		if err := m.RegisterContracts(ctx, moduleContracts); err != nil {
			return nil, fmt.Errorf("register contracts of module %s: %w", m.Name(), err)
		}
		logger.DebugContext(ctx, "initialized module", "module", m.Name(), "duration", time.Since(start))
	}
	if err := moduleContracts.Verify(); err != nil {
		return nil, fmt.Errorf("verify module contracts: %w", err)
	}

	// RequestID runs first so the logger and every error log see the same ID; Recoverer sits
	// inside the logger so a panic is still logged as a 500.
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(log.Middleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true, // the auth cookie must be sent cross-origin (e.g. without the Vite proxy)
		MaxAge:           300,
	}))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Every module's routes under /api; requireUser lets only signed-in users through (identity
	// checks the session).
	requireUser := web.RequireUser(func(ctx context.Context, token string) (uuid.UUID, error) {
		return moduleContracts.Identity.Authenticate(ctx, token)
	})
	var routeErr error
	r.Route("/api", func(api chi.Router) {
		// JSON like every other API response (chi answers in plain text by default).
		api.NotFound(func(w http.ResponseWriter, r *http.Request) {
			web.WriteError(w, r, http.StatusNotFound, "not found")
		})
		api.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			web.WriteError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		})
		for _, m := range modules {
			if err := m.RegisterHTTP(ctx, api, requireUser); err != nil {
				routeErr = errors.Join(routeErr, fmt.Errorf("register HTTP of module %s: %w", m.Name(), err))
			}
		}
	})
	if routeErr != nil {
		return nil, routeErr
	}

	// In production the built frontend is served next to the API (same origin, so the auth
	// cookie just works); in dev StaticDir is empty and Vite serves it.
	if cfg.StaticDir != "" {
		r.Handle("/*", web.SPA(cfg.StaticDir))
		logger.Info("serving frontend", "dir", cfg.StaticDir)
	}
	return &Svc{router: r, logger: logger}, nil
}

// Handler is the service's HTTP handler, e.g. for an httptest.Server.
func (s *Svc) Handler() http.Handler { return s.router }

// Run serves on addr (e.g. ":8080") until ctx is cancelled, then shuts down gracefully.
func (s *Svc) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Buffered so the goroutine can exit even when nobody reads the error (after a shutdown).
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Stop accepting connections and give in-flight requests up to 10s to finish.
		s.logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
