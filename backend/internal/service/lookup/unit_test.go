package lookup

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
)

func TestCreateUnit(t *testing.T) {
	f := &fakeQueries{}
	s := &Service{q: f}
	rpe, err := s.CreateLoadUnit(context.Background(), UnitInput{Name: " RPE ", MinValue: ptr(1.0), MaxValue: ptr(10.0)})
	if err != nil {
		t.Fatal(err)
	}
	if rpe.Name != "RPE" || *rpe.MaxValue != 10 {
		t.Errorf("load unit = %+v", rpe)
	}
	if _, err := s.CreateRepUnit(context.Background(), UnitInput{Name: "s"}); err != nil {
		t.Fatal(err)
	}
	if len(f.repUnits) != 1 || f.repUnits[0].MinValue != nil {
		t.Errorf("rep units = %+v, want one unbounded", f.repUnits)
	}
}

func TestCreateUnitRejects(t *testing.T) {
	tests := []struct {
		name     string
		in       UnitInput
		err      error // from the insert
		wantKind service.Kind
		wantMsg  string
	}{
		{"blank name", UnitInput{Name: "  "}, nil, service.KindInvalid, "name is required (max 30 characters)"},
		{"long name", UnitInput{Name: strings.Repeat("x", 31)}, nil, service.KindInvalid, "name is required (max 30 characters)"},
		{"negative bound", UnitInput{Name: "kg", MinValue: ptr(-1.0)}, nil, service.KindInvalid, "bounds must be between 0 and 99999"},
		{"bound too high", UnitInput{Name: "kg", MaxValue: ptr(100000.0)}, nil, service.KindInvalid, "bounds must be between 0 and 99999"},
		{"max below min", UnitInput{Name: "kg", MinValue: ptr(5.0), MaxValue: ptr(4.0)}, nil, service.KindInvalid, "max_value must not be lower than min_value"},
		{"duplicate", UnitInput{Name: "kg"}, &pgconn.PgError{Code: "23505"}, service.KindConflict, "this unit already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{q: &fakeQueries{err: tt.err}}
			_, err := s.CreateRepUnit(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
			_, err = s.CreateLoadUnit(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
		})
	}
}
