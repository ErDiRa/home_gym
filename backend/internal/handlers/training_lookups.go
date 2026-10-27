package handlers

import (
	"context"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	lookuprepos "github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

// Block kinds and the units of planned reps and load, which trainings are validated against.

func (h *Handler) ListBlockKinds(ctx context.Context, _ api.ListBlockKindsRequestObject) (api.ListBlockKindsResponseObject, error) {
	rows, err := list(ctx, h.lookups.BlockKinds, func(k lookuprepos.ListBlockKindsRow) api.BlockKind {
		return api.BlockKind{
			Code: k.Code, Name: k.Name, Description: k.Description, MinExercises: k.MinExercises, MaxExercises: k.MaxExercises,
			SharedSets: k.SharedSets, SharedReps: k.SharedReps, SharedLoad: k.SharedLoad, Scored: k.Scored,
		}
	})
	return api.ListBlockKinds200JSONResponse(rows), err
}

func (h *Handler) ListRepUnits(ctx context.Context, _ api.ListRepUnitsRequestObject) (api.ListRepUnitsResponseObject, error) {
	rows, err := list(ctx, h.lookups.RepUnits, repUnitToAPI)
	return api.ListRepUnits200JSONResponse(rows), err
}

func (h *Handler) ListLoadUnits(ctx context.Context, _ api.ListLoadUnitsRequestObject) (api.ListLoadUnitsResponseObject, error) {
	rows, err := list(ctx, h.lookups.LoadUnits, loadUnitToAPI)
	return api.ListLoadUnits200JSONResponse(rows), err
}

func (h *Handler) CreateRepUnit(ctx context.Context, req api.CreateRepUnitRequestObject) (api.CreateRepUnitResponseObject, error) {
	unit, err := h.lookups.CreateRepUnit(ctx, unitInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateRepUnit201JSONResponse(repUnitToAPI(lookuprepos.ListRepUnitsRow(unit))), nil
}

func (h *Handler) CreateLoadUnit(ctx context.Context, req api.CreateLoadUnitRequestObject) (api.CreateLoadUnitResponseObject, error) {
	unit, err := h.lookups.CreateLoadUnit(ctx, unitInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateLoadUnit201JSONResponse(loadUnitToAPI(lookuprepos.ListLoadUnitsRow(unit))), nil
}

func unitInputFromAPI(in api.UnitInput) lookup.UnitInput {
	return lookup.UnitInput{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue}
}

// repUnitToAPI maps a rep unit (CreateRepUnitRow converts to the list row).
func repUnitToAPI(u lookuprepos.ListRepUnitsRow) api.Unit {
	return api.Unit{Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue}
}

// loadUnitToAPI maps a load unit (CreateLoadUnitRow converts to the list row).
func loadUnitToAPI(u lookuprepos.ListLoadUnitsRow) api.LoadUnit {
	return api.LoadUnit{
		Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue, KgPerUnit: u.KgPerUnit, PercentOfMax: u.PercentOfMax,
	}
}
