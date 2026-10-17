package lookup

import (
	"context"
	"strings"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

// UnitInput is a new rep or load unit; nil bounds mean unbounded.
type UnitInput struct {
	Name     string   `json:"name"`
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

// CreateRepUnit adds a unit for planned reps, e.g. "s" or "m".
func (s *Service) CreateRepUnit(ctx context.Context, in UnitInput) (repos.CreateRepUnitRow, error) {
	if err := validateUnit(&in); err != nil {
		return repos.CreateRepUnitRow{}, err
	}
	unit, err := s.q.CreateRepUnit(ctx, repos.CreateRepUnitParams{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue})
	return unit, unitError(err)
}

// CreateLoadUnit adds a unit for planned load, e.g. "kg" or "RPE".
func (s *Service) CreateLoadUnit(ctx context.Context, in UnitInput) (repos.CreateLoadUnitRow, error) {
	if err := validateUnit(&in); err != nil {
		return repos.CreateLoadUnitRow{}, err
	}
	unit, err := s.q.CreateLoadUnit(ctx, repos.CreateLoadUnitParams{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue})
	return unit, unitError(err)
}

// validateUnit trims and validates a unit, in place.
func validateUnit(in *UnitInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || tooLong(in.Name, 30) {
		return service.Invalid("name is required (max 30 characters)")
	}
	for _, v := range []*float64{in.MinValue, in.MaxValue} {
		// Bounds must fit the NUMERIC(7, 2) columns, like the amounts they bound.
		if v != nil && (*v < 0 || *v >= maxAmount) {
			return service.Invalid("bounds must be between 0 and 99999")
		}
	}
	if in.MinValue != nil && in.MaxValue != nil && *in.MaxValue < *in.MinValue {
		return service.Invalid("max_value must not be lower than min_value")
	}
	return nil
}

// unitError maps a failed insert of a unit.
func unitError(err error) error {
	if isUniqueViolation(err) {
		return service.Conflict("this unit already exists", err)
	}
	return err
}
