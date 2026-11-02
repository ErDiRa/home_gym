//go:build integration

package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	nutritiondb "github.com/erdira/homegym/backend/modules/nutrition/adapters/db"
	"github.com/erdira/homegym/backend/modules/nutrition/app"
	"github.com/erdira/homegym/backend/tests/testdb"
)

func ptr[T any](v T) *T { return &v }

func date(s string) time.Time {
	d, _ := time.Parse(time.DateOnly, s)
	return d
}

func TestMacrosPerDay(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	repo := nutritiondb.NewRepo(pool)
	userID, other := testdb.User(t, pool), testdb.User(t, pool)

	if _, err := repo.SaveMacros(ctx, userID, app.Macros{Day: date("2026-10-27"), ProteinG: ptr(150.0), CarbsG: ptr(200.0)}); err != nil {
		t.Fatal(err)
	}
	// Saving the day again replaces it; the energy follows from the macros.
	m, err := repo.SaveMacros(ctx, userID, app.Macros{Day: date("2026-10-27"), ProteinG: ptr(160.0), FatG: ptr(70.0), Notes: "x"})
	if err != nil || m.Kcal != 160*4+70*9 || m.CarbsG != nil {
		t.Fatalf("replaced = %+v, %v", m, err)
	}
	if _, err := repo.SaveMacros(ctx, userID, app.Macros{Day: date("2026-10-29"), FatG: ptr(80.0)}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.Macros(ctx, userID, date("2026-10-27"), date("2026-10-28"))
	if err != nil || len(list) != 1 || *list[0].ProteinG != 160 || !list[0].Day.Equal(date("2026-10-27")) {
		t.Errorf("macros = %+v, %v", list, err)
	}
	if list, _ := repo.Macros(ctx, other, date("2026-10-01"), date("2026-10-31")); len(list) != 0 {
		t.Errorf("another user's macros = %+v", list)
	}

	// Deleting is soft: the day can be logged again.
	if err := repo.DeleteMacros(ctx, other, date("2026-10-27")); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("delete another user's = %v", err)
	}
	if err := repo.DeleteMacros(ctx, userID, date("2026-10-27")); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveMacros(ctx, userID, app.Macros{Day: date("2026-10-27"), CarbsG: ptr(300.0)}); err != nil {
		t.Errorf("log a deleted day again: %v", err)
	}
}

func TestBody(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	repo := nutritiondb.NewRepo(pool)
	userID, other := testdb.User(t, pool), testdb.User(t, pool)

	if p, err := repo.Profile(ctx, userID); p != nil || err != nil {
		t.Errorf("no profile = %+v, %v", p, err)
	}
	p := app.Profile{HeightCm: 180, BirthDate: date("1990-05-01"), Sex: app.Male, ActivityFactor: 1.3}
	if err := repo.SaveProfile(ctx, userID, p); err != nil {
		t.Fatal(err)
	}
	p.HeightCm = 181
	if err := repo.SaveProfile(ctx, userID, p); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.Profile(ctx, userID); err != nil || got.HeightCm != 181 || got.Sex != app.Male {
		t.Errorf("profile = %+v, %v", got, err)
	}

	first, err := repo.SaveWeight(ctx, userID, app.Weight{MeasuredOn: date("2026-10-20"), WeightKg: 81.2, BodyFatPercent: ptr(16.0)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveWeight(ctx, userID, app.Weight{MeasuredOn: date("2026-10-27"), WeightKg: 80.5}); err != nil {
		t.Fatal(err)
	}
	// The same day again replaces its weight and body fat.
	again, err := repo.SaveWeight(ctx, userID, app.Weight{MeasuredOn: date("2026-10-20"), WeightKg: 81, BodyFatPercent: ptr(15.5)})
	if err != nil || again.ID != first.ID || *again.BodyFatPercent != 15.5 {
		t.Errorf("same day = %+v, %v", again, err)
	}
	weights, err := repo.Weights(ctx, userID)
	if err != nil || len(weights) != 2 || weights[0].WeightKg != 80.5 || weights[0].BodyFatPercent != nil || weights[1].WeightKg != 81 {
		t.Errorf("weights = %+v, %v", weights, err)
	}
	if err := repo.DeleteWeight(ctx, other, first.ID); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("delete another user's = %v", err)
	}
	if err := repo.DeleteWeight(ctx, userID, first.ID); err != nil {
		t.Fatal(err)
	}
	if weights, _ := repo.Weights(ctx, userID); len(weights) != 1 {
		t.Errorf("after delete = %+v", weights)
	}
}
