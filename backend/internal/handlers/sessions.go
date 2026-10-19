package handlers

import (
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/training"
)

// ListSessions returns the user's session templates with their size, for the library.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	listOwn(w, r, h.trainings.ListSessions)
}

// GetSession returns one of the user's session templates with its blocks and exercises.
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "session")
	if !ok {
		return
	}
	s, err := h.trainings.GetSession(r.Context(), userID, id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// CreateSession saves a new session template with its blocks.
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	var in training.SessionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.trainings.CreateSession(r.Context(), userID, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// UpdateSession replaces a session template's fields and blocks; trainings keep their copies.
func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "session")
	if !ok {
		return
	}
	var in training.SessionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.trainings.UpdateSession(r.Context(), userID, id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// DeleteSession soft-deletes a session template; trainings keep their copies.
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := userAndID(w, r, "session")
	if !ok {
		return
	}
	if err := h.trainings.DeleteSession(r.Context(), userID, id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
