package handlers

import (
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/cycle"
	"github.com/erdira/homegym/backend/internal/service/training"
)

// ListCycles returns the user's cycles with their goal and training counts.
func (h *Handler) ListCycles(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.cycles.List)
}

// GetCycle returns one of the user's cycles with its goals and trainings.
func (h *Handler) GetCycle(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "cycle")
	if !ok {
		return
	}
	c, err := h.cycles.Get(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// CreateCycle saves a new cycle with its goals.
func (h *Handler) CreateCycle(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var in cycle.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.cycles.Create(r.Context(), userID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// UpdateCycle replaces a cycle's fields and goals.
func (h *Handler) UpdateCycle(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "cycle")
	if !ok {
		return
	}
	var in cycle.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.cycles.Update(r.Context(), userID, id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteCycle soft-deletes a cycle. ?trainings=delete deletes its trainings too; by default
// (keep) they stay and just leave the cycle.
func (h *Handler) DeleteCycle(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "cycle")
	if !ok {
		return
	}
	var withTrainings bool
	switch r.URL.Query().Get("trainings") {
	case "", "keep":
	case "delete":
		withTrainings = true
	default:
		writeError(w, r, http.StatusBadRequest, "trainings must be keep or delete")
		return
	}
	if err := h.cycles.Delete(r.Context(), userID, id, withTrainings); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// maxImportBody is the body limit of a cycle import: a 10-week cycle with ids is a few hundred KB.
const maxImportBody = 8 << 20

// ImportCycle saves a whole cycle with its trainings at once, or nothing if any part is invalid
// (a usecase of the training service, which writes the trainings).
func (h *Handler) ImportCycle(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var in training.CycleImportInput
	if !decodeJSONLimit(w, r, &in, maxImportBody) {
		return
	}
	c, err := h.trainings.ImportCycle(r.Context(), userID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
