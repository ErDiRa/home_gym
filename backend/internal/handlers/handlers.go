package handlers

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/database"
)

type Handler struct {
	pool    *pgxpool.Pool
	queries *database.Queries
	tokens  *auth.TokenManager

	// secureCookies marks the auth cookie Secure (HTTPS only); disable for local HTTP dev.
	secureCookies bool
}

func New(pool *pgxpool.Pool, tokens *auth.TokenManager, secureCookies bool) *Handler {
	return &Handler{pool: pool, queries: database.New(pool), tokens: tokens, secureCookies: secureCookies}
}

// inTx runs fn in a transaction, committing if it returns nil and rolling back otherwise.
func (h *Handler) inTx(ctx context.Context, fn func(q *database.Queries) error) error {
	return pgx.BeginFunc(ctx, h.pool, func(tx pgx.Tx) error {
		return fn(h.queries.WithTx(tx))
	})
}

// Routes registers the API routes on r:
//
//	/api/auth/*    login, logout and session restore
//	/api/client/*  endpoints for authenticated users
//	/api/admin/*   endpoints for admins only (separate accounts, not linked to users)
func (h *Handler) Routes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.tokens))
			r.Get("/auth/me", h.Me)
			r.With(auth.RequireUser(h.queries.UserExists)).Route("/client", h.clientRoutes)
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.RequireAdmin(h.queries.IsAdmin))
				h.adminRoutes(r)
			})
		})
	})
}
