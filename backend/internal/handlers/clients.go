package handlers

import (
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/client"
)

// Clients are the rows of the users table, managed by admins.

// ListClients returns all live clients with their training count and active training's name.
func (h *Handler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clients.List(r.Context())
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

// CreateClient adds a client with an initial password.
func (h *Handler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var in client.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.clients.Create(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// UpdateClient changes a client's name and email. The password can't be changed here.
func (h *Handler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "client")
	if !ok {
		return
	}
	var in client.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := h.clients.Update(r.Context(), id, in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// GetClient returns a client with their trainings, the active one first.
func (h *Handler) GetClient(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "client")
	if !ok {
		return
	}
	c, err := h.clients.Get(r.Context(), id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteClient soft-deletes a client.
func (h *Handler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "client")
	if !ok {
		return
	}
	if err := h.clients.Delete(r.Context(), id); err != nil {
		serviceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListClientExerciseRatings returns how a client rated exercises, for the exercise picker.
func (h *Handler) ListClientExerciseRatings(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "client")
	if !ok {
		return
	}
	ratings, err := h.clients.ExerciseRatings(r.Context(), id)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ratings)
}
