package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
)

// Block kinds and the units of planned reps and load are the catalog's lists: trainings are
// validated against their rows, so a new kind or unit needs no code change.

// unit is a rep or load unit as validation needs it.
type unit struct {
	name     string
	min, max *float64
}

// lookups holds the rows a training is validated against: the kinds by code and the live units
// by id.
type lookups struct {
	kinds     map[string]catalog.BlockKind
	repUnits  map[uuid.UUID]unit
	loadUnits map[uuid.UUID]unit
}

// lookups reads the block kinds and units from the catalog.
func (s *Service) lookups(ctx context.Context) (lookups, error) {
	kinds, err := s.catalog.BlockKinds(ctx)
	if err != nil {
		return lookups{}, err
	}
	units, err := s.catalog.Units(ctx)
	if err != nil {
		return lookups{}, err
	}
	lk := lookups{
		kinds:     make(map[string]catalog.BlockKind, len(kinds)),
		repUnits:  liveUnits(units.Rep),
		loadUnits: liveUnits(units.Load),
	}
	for _, k := range kinds {
		lk.kinds[k.Code] = k
	}
	return lk, nil
}

func liveUnits(units []catalog.Unit) map[uuid.UUID]unit {
	out := make(map[uuid.UUID]unit, len(units))
	for _, u := range units {
		if !u.Deleted {
			out[u.ID] = unit{name: u.Name, min: u.MinValue, max: u.MaxValue}
		}
	}
	return out
}

// checkExercises fails with "unknown exercise" unless every exercise of the sessions is one of the
// user's live exercises (the catalog's).
func (s *Service) checkExercises(ctx context.Context, userID uuid.UUID, sessions ...SessionInput) error {
	var ids []uuid.UUID
	for _, session := range sessions {
		for _, b := range session.Blocks {
			for _, e := range b.Exercises {
				ids = append(ids, e.ExerciseID)
			}
		}
	}
	exercises, err := s.catalog.Exercises(ctx, userID, ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if e, ok := exercises[id]; !ok || e.Deleted {
			return errs.Invalid("unknown exercise")
		}
	}
	return nil
}
