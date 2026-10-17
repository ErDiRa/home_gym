package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/database"
)

// Block kinds and the units of planned reps and load are lookup tables: trainings are validated
// against their rows, so a new kind or unit needs no code change.

func (h *Handler) ListBlockKinds(w http.ResponseWriter, r *http.Request) {
	kinds, err := h.queries.ListBlockKinds(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, kinds)
}

func (h *Handler) ListRepUnits(w http.ResponseWriter, r *http.Request) {
	units, err := h.queries.ListRepUnits(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, units)
}

func (h *Handler) ListLoadUnits(w http.ResponseWriter, r *http.Request) {
	units, err := h.queries.ListLoadUnits(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, units)
}

type createUnitRequest struct {
	Name     string   `json:"name"`
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

func (h *Handler) CreateRepUnit(w http.ResponseWriter, r *http.Request) {
	createUnit(w, r, func(ctx context.Context, req createUnitRequest) (any, error) {
		return h.queries.CreateRepUnit(ctx, database.CreateRepUnitParams{
			Name: req.Name, MinValue: req.MinValue, MaxValue: req.MaxValue,
		})
	})
}

func (h *Handler) CreateLoadUnit(w http.ResponseWriter, r *http.Request) {
	createUnit(w, r, func(ctx context.Context, req createUnitRequest) (any, error) {
		return h.queries.CreateLoadUnit(ctx, database.CreateLoadUnitParams{
			Name: req.Name, MinValue: req.MinValue, MaxValue: req.MaxValue,
		})
	})
}

// createUnit validates a unit body and stores it with create, which differs per unit table.
func createUnit(w http.ResponseWriter, r *http.Request, create func(context.Context, createUnitRequest) (any, error)) {
	var req createUnitRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || tooLong(req.Name, 30) {
		writeError(w, r, http.StatusBadRequest, "name is required (max 30 characters)")
		return
	}
	for _, v := range []*float64{req.MinValue, req.MaxValue} {
		if v != nil && (*v < 0 || *v >= maxAmount) {
			writeError(w, r, http.StatusBadRequest, "bounds must be between 0 and 99999")
			return
		}
	}
	if req.MinValue != nil && req.MaxValue != nil && *req.MaxValue < *req.MinValue {
		writeError(w, r, http.StatusBadRequest, "max_value must not be lower than min_value")
		return
	}
	unit, err := create(r.Context(), req)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			writeError(w, r, http.StatusConflict, "this unit already exists")
			return
		}
		serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, unit)
}

// unit is a rep or load unit as validation needs it.
type unit struct {
	name     string
	min, max *float64
}

// trainingLookups holds the rows a training body is validated against.
type trainingLookups struct {
	kinds     map[string]database.ListBlockKindsRow
	repUnits  map[uuid.UUID]unit
	loadUnits map[uuid.UUID]unit
}

func loadTrainingLookups(ctx context.Context, q *database.Queries) (trainingLookups, error) {
	kinds, err := q.ListBlockKinds(ctx)
	if err != nil {
		return trainingLookups{}, err
	}
	repUnits, err := q.ListRepUnits(ctx)
	if err != nil {
		return trainingLookups{}, err
	}
	loadUnits, err := q.ListLoadUnits(ctx)
	if err != nil {
		return trainingLookups{}, err
	}
	lk := trainingLookups{
		kinds:     make(map[string]database.ListBlockKindsRow, len(kinds)),
		repUnits:  make(map[uuid.UUID]unit, len(repUnits)),
		loadUnits: make(map[uuid.UUID]unit, len(loadUnits)),
	}
	for _, k := range kinds {
		lk.kinds[k.Code] = k
	}
	for _, u := range repUnits {
		lk.repUnits[u.ID] = unit{name: u.Name, min: u.MinValue, max: u.MaxValue}
	}
	for _, u := range loadUnits {
		lk.loadUnits[u.ID] = unit{name: u.Name, min: u.MinValue, max: u.MaxValue}
	}
	return lk, nil
}
