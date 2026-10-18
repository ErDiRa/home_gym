package handlers

import (
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/training"
)

// ListTrainings returns the user's trainings with their size, for the overview.
func (h *Handler) ListTrainings(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.trainings.List)
}

// GetTraining returns one of the user's trainings with its units, blocks and exercises.
func (h *Handler) GetTraining(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "training")
	if !ok {
		return
	}
	t, err := h.trainings.Get(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// GetActiveTraining returns the user's active training with its whole tree, or null.
func (h *Handler) GetActiveTraining(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	t, err := h.trainings.Active(r.Context(), userID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// CreateTraining saves a new training with its whole tree.
func (h *Handler) CreateTraining(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var in training.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.trainings.Create(r.Context(), userID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// UpdateTraining replaces a training's fields and tree.
func (h *Handler) UpdateTraining(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "training")
	if !ok {
		return
	}
	var in training.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.trainings.Update(r.Context(), userID, id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// DuplicateTraining copies a training as the next phase, inactive.
func (h *Handler) DuplicateTraining(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "training")
	if !ok {
		return
	}
	t, err := h.trainings.Duplicate(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// DeleteTraining soft-deletes a training.
func (h *Handler) DeleteTraining(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "training")
	if !ok {
		return
	}
	if err := h.trainings.Delete(r.Context(), userID, id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
