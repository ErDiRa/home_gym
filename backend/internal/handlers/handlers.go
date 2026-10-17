package handlers

import (
	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/internal/auth"
)

// Handler adapts the services to HTTP: it decodes requests, calls a service and writes its
// result or error as JSON.
type Handler struct {
	trainings TrainingService
	clients   ClientService
	exercises ExerciseService
	lookups   LookupService
	feedback  FeedbackService
	accounts  AccountService
	tokens    *auth.TokenManager

	// secureCookies marks the auth cookie Secure (HTTPS only); disable for local HTTP dev.
	secureCookies bool
}

// New builds the handlers; call Routes to register them.
func New(s Services, tokens *auth.TokenManager, secureCookies bool) *Handler {
	return &Handler{
		trainings: s.Trainings,
		clients:   s.Clients,
		exercises: s.Exercises,
		lookups:   s.Lookups,
		feedback:  s.Feedback,
		accounts:  s.Accounts,
		tokens:    tokens, secureCookies: secureCookies,
	}
}

// Routes registers the API routes on r:
//
//	/api/client/*  endpoints for authenticated users (the client app)
//	/api/admin/*   endpoints for admins only (separate accounts, not linked to users)
//
// Each audience has its own auth/login, auth/logout (public) and auth/me (guarded); a login
// only accepts accounts of its own audience, so neither app reveals the other's accounts.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Route("/client", func(r chi.Router) {
			r.Post("/auth/login", h.LoginUser)
			r.Post("/auth/logout", h.Logout)
			r.Group(func(r chi.Router) {
				r.Use(auth.Middleware(h.tokens), auth.RequireUser(h.accounts.UserExists))
				r.Get("/auth/me", h.Me)
				h.clientRoutes(r)
			})
		})
		r.Route("/admin", func(r chi.Router) {
			r.Post("/auth/login", h.LoginAdmin)
			r.Post("/auth/logout", h.Logout)
			r.Group(func(r chi.Router) {
				r.Use(auth.Middleware(h.tokens), auth.RequireAdmin(h.accounts.IsAdmin))
				r.Get("/auth/me", h.Me)
				h.adminRoutes(r)
			})
		})
	})
}
