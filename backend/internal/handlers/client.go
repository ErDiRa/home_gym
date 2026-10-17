package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/internal/service/feedback"
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

// ListAssignedTrainings returns the trainings the current user follows (their active one),
// each with its units, blocks and exercises.
func (h *Handler) ListAssignedTrainings(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	trainings, err := h.trainings.Assigned(r.Context(), userID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, trainings)
}

// UpsertFeedback stores the current user's rating (1-5) and notes for an exercise, replacing
// any earlier rating of theirs.
func (h *Handler) UpsertFeedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	exerciseID, ok := idParam(w, r, "exercise")
	if !ok {
		return
	}
	var in feedback.RatingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	f, err := h.feedback.Rate(r.Context(), userID, exerciseID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// ExerciseStats lists the exercises the current user has logged, most performed first, with
// their rating of each.
func (h *Handler) ExerciseStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	stats, err := h.feedback.Stats(r.Context(), userID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
