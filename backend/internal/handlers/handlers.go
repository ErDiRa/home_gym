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
			r.With(auth.RequireUser(h.accounts.UserExists)).Route("/client", h.clientRoutes)
			r.Route("/admin", func(r chi.Router) {
				r.Use(auth.RequireAdmin(h.accounts.IsAdmin))
				h.adminRoutes(r)
			})
		})
	})
}
