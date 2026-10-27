package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/auth"
)

// Handler implements the API of api/openapi.yaml (api.StrictServerInterface, generated from
// it): it maps the decoded requests to service calls and their results to the API models.
// Failures are returned as errors and answered by serviceError.
type Handler struct {
	trainings TrainingService
	cycles    CycleService
	exercises ExerciseService
	lookups   LookupService
	feedback  FeedbackService
	maxes     MaxService
	workouts  WorkoutService
	whoop     WhoopService
	accounts  AccountService
	tokens    *auth.TokenManager

	// secureCookies marks the auth cookie Secure (HTTPS only); disable for local HTTP dev.
	secureCookies bool
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New builds the handlers; call Routes to register them.
func New(s Services, tokens *auth.TokenManager, secureCookies bool) *Handler {
	return &Handler{
		trainings: s.Trainings,
		cycles:    s.Cycles,
		exercises: s.Exercises,
		lookups:   s.Lookups,
		feedback:  s.Feedback,
		maxes:     s.Maxes,
		workouts:  s.Workouts,
		whoop:     s.Whoop,
		accounts:  s.Accounts,
		tokens:    tokens, secureCookies: secureCookies,
	}
}

// Routes registers the API under /api. Login and logout are public; everything else needs a
// signed-in user and acts on that user's own data only.
func (h *Handler) Routes(r chi.Router) {
	h.routes(r, func(next http.Handler) http.Handler {
		return auth.Middleware(h.tokens)(auth.RequireUser(h.accounts.UserExists)(next))
	})
}

// routes registers the generated routes with authenticate guarding the non-public ones (a fake
// in tests).
func (h *Handler) routes(r chi.Router, authenticate func(http.Handler) http.Handler) {
	r.Route("/api", func(r chi.Router) {
		// JSON like every other API response (chi answers in plain text by default).
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, r, http.StatusNotFound, "not found")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		})
		r.Use(limitBody)
		strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{requireUser(authenticate)}, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  bodyError,
			ResponseErrorHandlerFunc: serviceError,
		})
		api.HandlerWithOptions(strict, api.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: paramError})
	})
}

// publicOperations are the operations the spec marks `security: []`.
var publicOperations = map[string]bool{"Login": true, "Logout": true}

// requireUser runs every operation but the public ones behind authenticate, which answers 401
// itself when there is no valid session.
func requireUser(authenticate func(http.Handler) http.Handler) api.StrictMiddlewareFunc {
	return func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		if publicOperations[operationID] {
			return next
		}
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
			authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response, err = next(r.Context(), w, r, request)
			})).ServeHTTP(w, r)
			return response, err
		}
	}
}
