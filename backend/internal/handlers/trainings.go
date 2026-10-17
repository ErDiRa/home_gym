package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training"
)

// ListTrainings returns all trainings with their client, for the admin overview.
func (h *Handler) ListTrainings(w http.ResponseWriter, r *http.Request) {
	trainings, err := h.trainings.List(r.Context())
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, trainings)
}

// GetTraining returns a training with its units, blocks and exercises.
func (h *Handler) GetTraining(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "training")
	if !ok {
		return
	}
	t, err := h.trainings.Get(r.Context(), id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// CreateTraining saves a new training with its whole tree.
func (h *Handler) CreateTraining(w http.ResponseWriter, r *http.Request) {
	var in training.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.trainings.Create(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// UpdateTraining replaces a training's fields and tree; it can also move the training to
// another client.
func (h *Handler) UpdateTraining(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "training")
	if !ok {
		return
	}
	var in training.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	t, err := h.trainings.Update(r.Context(), id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type duplicateTrainingRequest struct {
	ClientID uuid.UUID `json:"client_id"`
}

// DuplicateTraining copies a training for a client (the same or another one), inactive.
func (h *Handler) DuplicateTraining(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "training")
	if !ok {
		return
	}
	var req duplicateTrainingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	t, err := h.trainings.Duplicate(r.Context(), id, req.ClientID)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// DeleteTraining soft-deletes a training.
func (h *Handler) DeleteTraining(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "training")
	if !ok {
		return
	}
	if err := h.trainings.Delete(r.Context(), id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
