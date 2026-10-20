package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/internal/auth"
)

// Handler adapts the services to HTTP: it decodes requests, calls a service and writes its
// result or error as JSON.
type Handler struct {
	trainings TrainingService
	exercises ExerciseService
	lookups   LookupService
	feedback  FeedbackService
	workouts  WorkoutService
	accounts  AccountService
	tokens    *auth.TokenManager

	// secureCookies marks the auth cookie Secure (HTTPS only); disable for local HTTP dev.
	secureCookies bool
}

// New builds the handlers; call Routes to register them.
func New(s Services, tokens *auth.TokenManager, secureCookies bool) *Handler {
	return &Handler{
		trainings: s.Trainings,
		exercises: s.Exercises,
		lookups:   s.Lookups,
		feedback:  s.Feedback,
		workouts:  s.Workouts,
		accounts:  s.Accounts,
		tokens:    tokens, secureCookies: secureCookies,
	}
}

// Routes registers the API routes on r. auth/login and auth/logout are public; everything
// else needs a signed-in user and acts on that user's own data only.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		// JSON like every other API response (chi answers in plain text by default).
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, r, http.StatusNotFound, "not found")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		})
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.tokens), auth.RequireUser(h.accounts.UserExists))
			r.Get("/auth/me", h.Me)

			r.Get("/exercises", h.ListExercises)
			r.Post("/exercises", h.CreateExercise)
			r.Get("/exercises/{id}", h.GetExercise)
			r.Put("/exercises/{id}", h.UpdateExercise)
			r.Post("/exercises/{id}/feedback", h.RateExercise)
			r.Get("/exercise-ratings", h.ListExerciseRatings)
			r.Get("/stats/exercises", h.ExerciseStats)

			r.Get("/joints", h.ListJoints)
			r.Get("/muscle-groups", h.ListMuscleGroups)
			r.Get("/focuses", h.ListFocuses)
			r.Post("/focuses", h.CreateFocus)
			r.Get("/block-kinds", h.ListBlockKinds)
			r.Get("/rep-units", h.ListRepUnits)
			r.Post("/rep-units", h.CreateRepUnit)
			r.Get("/load-units", h.ListLoadUnits)
			r.Post("/load-units", h.CreateLoadUnit)

			r.Get("/sessions", h.ListSessions)
			r.Post("/sessions", h.CreateSession)
			r.Get("/sessions/{id}", h.GetSession)
			r.Put("/sessions/{id}", h.UpdateSession)
			r.Delete("/sessions/{id}", h.DeleteSession)

			r.Get("/trainings", h.ListTrainings)
			r.Post("/trainings", h.CreateTraining)
			r.Get("/trainings/{id}", h.GetTraining)
			r.Put("/trainings/{id}", h.UpdateTraining)
			r.Delete("/trainings/{id}", h.DeleteTraining)
			r.Post("/trainings/{id}/duplicate", h.DuplicateTraining)

			r.Get("/week", h.GetWeek)
			r.Get("/workouts", h.ListWorkouts)
			r.Post("/workouts", h.StartWorkout)
			r.Get("/workouts/{id}", h.GetWorkout)
			r.Delete("/workouts/{id}", h.DeleteWorkout)
			r.Post("/workouts/{id}/finish", h.FinishWorkout)
			r.Put("/workouts/{id}/sets", h.LogSet)
			r.Delete("/workouts/{id}/sets/{setID}", h.DeleteSet)
		})
	})
}
