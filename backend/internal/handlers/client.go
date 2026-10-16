package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/stefankarlin/my-accessories/backend/internal/database"
)

// clientRoutes are mounted under /api/client for any authenticated user.
func (h *Handler) clientRoutes(r chi.Router) {
	r.Get("/exercises", h.ListExercises)
	r.Get("/exercises/{id}", h.GetExercise)
	r.Get("/joints", h.ListJoints)
	r.Get("/muscle-groups", h.ListMuscleGroups)
	r.Get("/focuses", h.ListFocuses)
	r.Get("/trainings", h.ListAssignedTrainings)
	r.Post("/exercises/{id}/feedback", h.UpsertFeedback)
	r.Get("/stats/exercises", h.ExerciseStats)
}

// ListAssignedTrainings returns the trainings assigned to the current user. Not implemented yet.
func (h *Handler) ListAssignedTrainings(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w)
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
