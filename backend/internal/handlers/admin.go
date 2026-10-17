package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/internal/service/exercise"
	"github.com/erdira/homegym/backend/internal/service/lookup"
)

// adminRoutes are mounted under /api/admin behind auth.RequireAdmin.
func (h *Handler) adminRoutes(r chi.Router) {
	r.Post("/exercises", h.CreateExercise)
	r.Get("/exercises", h.ListExercises)
	r.Get("/exercises/{id}", h.GetExercise)
	r.Put("/exercises/{id}", h.UpdateExercise)
	r.Get("/joints", h.ListJoints)
	r.Get("/muscle-groups", h.ListMuscleGroups)
	r.Get("/focuses", h.ListFocuses)
	r.Post("/focuses", h.CreateFocus)
	r.Get("/block-kinds", h.ListBlockKinds)
	r.Get("/rep-units", h.ListRepUnits)
	r.Post("/rep-units", h.CreateRepUnit)
	r.Get("/load-units", h.ListLoadUnits)
	r.Post("/load-units", h.CreateLoadUnit)
	r.Get("/trainings", h.ListTrainings)
	r.Post("/trainings", h.CreateTraining)
	r.Get("/trainings/{id}", h.GetTraining)
	r.Put("/trainings/{id}", h.UpdateTraining)
	r.Delete("/trainings/{id}", h.DeleteTraining)
	r.Post("/trainings/{id}/duplicate", h.DuplicateTraining)
	r.Get("/clients", h.ListClients)
	r.Post("/clients", h.CreateClient)
	r.Get("/clients/{id}", h.GetClient)
	r.Put("/clients/{id}", h.UpdateClient)
	r.Delete("/clients/{id}", h.DeleteClient)
	r.Get("/clients/{id}/exercise-ratings", h.ListClientExerciseRatings)
}

func (h *Handler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	var in exercise.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.exercises.Create(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "exercise")
	if !ok {
		return
	}
	var in exercise.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.exercises.Update(r.Context(), id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// GetExercise returns one exercise with the ids of its links, for filling the exercise form.
// It is shared by the admin and client routes.
func (h *Handler) GetExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "exercise")
	if !ok {
		return
	}
	e, err := h.exercises.Get(r.Context(), id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// ListExercises returns all exercises with the names of their links. Shared by the admin and
// client routes.
func (h *Handler) ListExercises(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.exercises.List)
}

func (h *Handler) ListJoints(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.Joints)
}

func (h *Handler) ListMuscleGroups(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.MuscleGroups)
}

func (h *Handler) ListFocuses(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.Focuses)
}

func (h *Handler) CreateFocus(w http.ResponseWriter, r *http.Request) {
	var in lookup.FocusInput
	if !decodeJSON(w, r, &in) {
		return
	}
	focus, err := h.lookups.CreateFocus(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, focus)
}
