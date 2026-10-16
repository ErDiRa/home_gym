package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/stefankarlin/my-accessories/backend/internal/database"
)

// adminRoutes are mounted under /api/admin behind auth.RequireAdmin.
func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/exercises", h.CreateExercise)
	r.Get("/exercises", h.ListExercises)
	r.Post("/training-plans", h.CreateTrainingPlan)
	r.Post("/users/{userID}/trainings", h.AssignTraining)
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

// CreateTrainingPlan is not implemented yet.
func (h *Handler) CreateTrainingPlan(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w)
}

// AssignTraining assigns a training plan to a user. Not implemented yet.
func (h *Handler) AssignTraining(w http.ResponseWriter, _ *http.Request) {
	notImplemented(w)
}
