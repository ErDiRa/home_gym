package app

import (
	"context"
	"errors"
	"strings"

	"github.com/erdira/homegym/backend/common/errs"
)

// maxAmount is the exclusive upper bound of an amount value (NUMERIC(7, 2) columns).
const maxAmount = 100000

// FocusInput is a new focus, e.g. "Hip mobility" with the alias "HIP".
type FocusInput struct {
	Name  string
	Alias string
}

// CreateFocus adds a focus for exercises.
func (s *Service) CreateFocus(ctx context.Context, in FocusInput) (Focus, error) {
	in.Name, in.Alias = strings.TrimSpace(in.Name), strings.TrimSpace(in.Alias)
	if in.Name == "" || tooLong(in.Name, 60) {
		return Focus{}, errs.Invalid("name is required (max 60 characters)")
	}
	if in.Alias == "" || tooLong(in.Alias, 10) {
		return Focus{}, errs.Invalid("alias is required (max 10 characters)")
	}
	focus, err := s.lookups.CreateFocus(ctx, in)
	if errors.Is(err, ErrDuplicate) {
		return Focus{}, errs.Conflict("this focus already exists", err)
	}
	return focus, err
}

// UnitInput is a new rep or load unit; nil bounds mean unbounded.
type UnitInput struct {
	Name     string
	MinValue *float64
	MaxValue *float64
}

// CreateRepUnit adds a unit for planned reps, e.g. "s" or "m".
func (s *Service) CreateRepUnit(ctx context.Context, in UnitInput) (Unit, error) {
	if err := validateUnit(&in); err != nil {
		return Unit{}, err
	}
	unit, err := s.lookups.CreateRepUnit(ctx, in)
	return unit, unitError(err)
}

// CreateLoadUnit adds a unit for planned load, e.g. "kg" or "RPE".
func (s *Service) CreateLoadUnit(ctx context.Context, in UnitInput) (Unit, error) {
	if err := validateUnit(&in); err != nil {
		return Unit{}, err
	}
	unit, err := s.lookups.CreateLoadUnit(ctx, in)
	return unit, unitError(err)
}

// validateUnit trims and validates a unit, in place.
func validateUnit(in *UnitInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || tooLong(in.Name, 30) {
		return errs.Invalid("name is required (max 30 characters)")
	}
	for _, v := range []*float64{in.MinValue, in.MaxValue} {
		// Bounds must fit the NUMERIC(7, 2) columns, like the amounts they bound.
		if v != nil && (*v < 0 || *v >= maxAmount) {
			return errs.Invalid("bounds must be between 0 and 99999")
		}
	}
	if in.MinValue != nil && in.MaxValue != nil && *in.MaxValue < *in.MinValue {
		return errs.Invalid("max_value must not be lower than min_value")
	}
	return nil
}

func unitError(err error) error {
	if errors.Is(err, ErrDuplicate) {
		return errs.Conflict("this unit already exists", err)
	}
	return err
}
