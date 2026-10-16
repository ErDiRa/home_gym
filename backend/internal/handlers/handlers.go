package handlers

import (
	"github.com/go-chi/chi/v5"

	"github.com/stefankarlin/my-accessories/backend/internal/auth"
	"github.com/stefankarlin/my-accessories/backend/internal/database"
)

type Handler struct {
	queries *database.Queries
	tokens  *auth.TokenManager

	// secureCookies marks the auth cookie Secure (HTTPS only); disable for local HTTP dev.
	secureCookies bool
}

func New(queries *database.Queries, tokens *auth.TokenManager, secureCookies bool) *Handler {
	return &Handler{queries: queries, tokens: tokens, secureCookies: secureCookies}
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
			r.With(auth.RequireUser).Route("/client", h.clientRoutes)
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.RequireAdmin(h.queries.IsAdmin))
				h.adminRoutes(r)
			})
		})
	})
}
