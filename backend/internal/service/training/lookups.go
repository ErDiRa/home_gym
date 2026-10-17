package training

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// Block kinds and the units of planned reps and load are lookup tables: trainings are validated
// against their rows, so a new kind or unit needs no code change.

// unit is a rep or load unit as validation needs it.
type unit struct {
	name     string
	min, max *float64
}

// lookups holds the rows a training is validated against.
type lookups struct {
	kinds     map[string]repos.ListBlockKindsRow
	repUnits  map[uuid.UUID]unit
	loadUnits map[uuid.UUID]unit
}

// loadLookups reads the block kinds and units into maps keyed by code/id.
func loadLookups(ctx context.Context, q Queries) (lookups, error) {
	kinds, err := q.ListBlockKinds(ctx)
	if err != nil {
		return lookups{}, err
	}
	repUnits, err := q.ListRepUnits(ctx)
	if err != nil {
		return lookups{}, err
	}
	loadUnits, err := q.ListLoadUnits(ctx)
	if err != nil {
		return lookups{}, err
	}
	lk := lookups{
		kinds:     make(map[string]repos.ListBlockKindsRow, len(kinds)),
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
