package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

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

// Routes registers the API routes on r.
func (h *Handler) Routes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", h.Login)
		r.Post("/auth/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(h.tokens))
			r.Get("/auth/me", h.Me)
			r.Post("/exercises", h.CreateExercise)
			r.Get("/exercises", h.ListExercises)
			r.Post("/exercises/{id}/feedback", h.UpsertFeedback)
			r.Get("/stats/exercises", h.ExerciseStats)
		})
	})
}

type createExerciseRequest struct {
	Name        string `json:"name"`
	Joint       string `json:"joint"`
	MuscleGroup string `json:"muscle_group"`
	Description string `json:"description"`
}

func (h *Handler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	var req createExerciseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Joint == "" || req.MuscleGroup == "" {
		writeError(w, http.StatusBadRequest, "name, joint and muscle_group are required")
		return
	}

	exercise, err := h.queries.CreateExercise(r.Context(), database.CreateExerciseParams{
		Name:        req.Name,
		Joint:       req.Joint,
		MuscleGroup: req.MuscleGroup,
		Description: req.Description,
	})
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, exercise)
}

func (h *Handler) ListExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.queries.ListExercises(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, exercises)
}

type feedbackRequest struct {
	Rating int32  `json:"rating"`
	Notes  string `json:"notes"`
}

func (h *Handler) UpsertFeedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	exerciseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid exercise id")
		return
	}

	var req feedbackRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	feedback, err := h.queries.UpsertExerciseFeedback(r.Context(), database.UpsertExerciseFeedbackParams{
		UserID:     userID,
		ExerciseID: exerciseID,
		Rating:     req.Rating,
		Notes:      req.Notes,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
			writeError(w, http.StatusNotFound, "user or exercise not found")
			return
		}
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, feedback)
}

func (h *Handler) ExerciseStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	stats, err := h.queries.GetExerciseStats(r.Context(), userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
