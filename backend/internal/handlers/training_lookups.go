package handlers

import (
	"context"
	"net/http"

	"github.com/erdira/homegym/backend/internal/service/lookup"
)

// Block kinds and the units of planned reps and load, which trainings are validated against.

func (h *Handler) ListBlockKinds(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.BlockKinds)
}

func (h *Handler) ListRepUnits(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.RepUnits)
}

func (h *Handler) ListLoadUnits(w http.ResponseWriter, r *http.Request) {
	list(w, r, h.lookups.LoadUnits)
}

func (h *Handler) CreateRepUnit(w http.ResponseWriter, r *http.Request) {
	var in lookup.UnitInput
	if !decodeJSON(w, r, &in) {
		return
	}
	unit, err := h.lookups.CreateRepUnit(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, unit)
}

func (h *Handler) CreateLoadUnit(w http.ResponseWriter, r *http.Request) {
	var in lookup.UnitInput
	if !decodeJSON(w, r, &in) {
		return
	}
	unit, err := h.lookups.CreateLoadUnit(r.Context(), in)
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, unit)
}

// list answers a GET with the rows fetch returns.
func list[T any](w http.ResponseWriter, r *http.Request, fetch func(context.Context) ([]T, error)) {
	rows, err := fetch(r.Context())
	if err != nil {
		serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
