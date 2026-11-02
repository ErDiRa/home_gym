package app

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	tracking "github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

// fakeRepo keeps one user's data.
type fakeRepo struct {
	profile *Profile
	weights []Weight // newest first
	macros  []Macros
	saved   []Macros
}

func (f *fakeRepo) Profile(context.Context, uuid.UUID) (*Profile, error) { return f.profile, nil }

func (f *fakeRepo) SaveProfile(_ context.Context, _ uuid.UUID, p Profile) error {
	f.profile = &p
	return nil
}

func (f *fakeRepo) Weights(context.Context, uuid.UUID) ([]Weight, error) { return f.weights, nil }

func (f *fakeRepo) SaveWeight(_ context.Context, _ uuid.UUID, w Weight) (Weight, error) {
	return w, nil
}

func (f *fakeRepo) DeleteWeight(context.Context, uuid.UUID, uuid.UUID) error { return ErrNotFound }

func (f *fakeRepo) Macros(context.Context, uuid.UUID, time.Time, time.Time) ([]Macros, error) {
	return f.macros, nil
}

func (f *fakeRepo) SaveMacros(_ context.Context, _ uuid.UUID, m Macros) (Macros, error) {
	f.saved = append(f.saved, m)
	return m, nil
}

func (f *fakeRepo) DeleteMacros(context.Context, uuid.UUID, time.Time) error { return ErrNotFound }

// fakeContracts answers for the tracking and whoop modules.
type fakeContracts struct {
	workouts []tracking.FinishedWorkout
	energy   map[string]float64
}

func (f *fakeContracts) FinishedWorkouts(context.Context, uuid.UUID, time.Time, time.Time) ([]tracking.FinishedWorkout, error) {
	return f.workouts, nil
}

func (f *fakeContracts) DailyEnergy(context.Context, uuid.UUID, time.Time, time.Time, *time.Location) (map[string]float64, error) {
	return f.energy, nil
}

func newTestService() (*Service, *fakeRepo, *fakeContracts) {
	repo, c := &fakeRepo{}, &fakeContracts{energy: map[string]float64{}}
	s := NewService(repo, c, c)
	s.now = func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) }
	return s, repo, c
}

func wantKind(t *testing.T, err error, kind errs.Kind) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Kind != kind {
		t.Errorf("err = %v, want an errs.Error of kind %d", err, kind)
	}
}

func date(s string) time.Time {
	d, _ := time.Parse(time.DateOnly, s)
	return d
}

func ptr[T any](v T) *T { return &v }

func TestRestingEnergy(t *testing.T) {
	day := date("2026-11-02")
	p := Profile{HeightCm: 180, BirthDate: date("1990-11-03"), Sex: Male, ActivityFactor: 1.3}
	// Mifflin-St Jeor: 10 × 80 + 6.25 × 180 - 5 × 35 (the birthday is tomorrow) + 5.
	if got := restingEnergy(p, 80, nil, day); got != 1755 {
		t.Errorf("male = %v", got)
	}
	p.Sex = Female
	if got := restingEnergy(p, 80, nil, day); got != 1589 {
		t.Errorf("female = %v", got)
	}
	// Katch-McArdle from the lean mass (80 kg at 15 %: 68 kg).
	if got := restingEnergy(p, 80, ptr(15.0), day); math.Abs(got-1838.8) > 0.01 {
		t.Errorf("lean mass = %v", got)
	}
}

func TestTarget(t *testing.T) {
	day := date("2026-11-02")
	p := Profile{HeightCm: 180, BirthDate: date("1990-11-03"), Sex: Male, ActivityFactor: 1.3}

	// A rest day: 1755 × 1.3; protein 2 g/kg, fat 0.9 g/kg, the rest carbs.
	rest := target(p, 80, nil, day, nil, nil)
	if rest != (Target{Kcal: 2282, ProteinG: 160, FatG: 72, CarbsG: 249, Source: SourceEstimate, WeightKg: 80}) {
		t.Errorf("rest day = %+v", rest)
	}
	// An hour of training adds 4 × 80 kcal; a workout left open all day counts 2.5 hours.
	if got := target(p, 80, nil, day, nil, []time.Duration{time.Hour}); got.Kcal != 2602 || got.CarbsG != 329 {
		t.Errorf("training day = %+v", got)
	}
	if got := target(p, 80, nil, day, nil, []time.Duration{10 * time.Hour}); got.Kcal != 3082 {
		t.Errorf("an open workout = %+v", got)
	}
	// WHOOP's energy replaces the estimate; fat is at least a fifth of it.
	got := target(p, 80, nil, day, ptr(16736.0), []time.Duration{time.Hour})
	if got.Kcal != 4000 || got.Source != SourceWhoop || got.FatG != 89 || got.CarbsG != 640 {
		t.Errorf("WHOOP = %+v", got)
	}
}

func TestDays(t *testing.T) {
	s, repo, c := newTestService()
	userID := uuid.New()

	// Without a profile there are no targets, but the macros and workouts.
	repo.macros = []Macros{{Day: date("2026-10-27"), ProteinG: ptr(150.0), Kcal: 600}}
	// Finished at 00:30 local time on the 28th (23:30 UTC on the 27th).
	c.workouts = []tracking.FinishedWorkout{{ID: uuid.New(), SessionName: "Session 1",
		StartedAt: time.Date(2026, 10, 27, 22, 30, 0, 0, time.UTC), CompletedAt: time.Date(2026, 10, 27, 23, 30, 0, 0, time.UTC)}}
	days, err := s.Days(context.Background(), userID, "2026-10-26", "2026-11-01", "Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 7 || days[1].Macros == nil || *days[1].Macros.ProteinG != 150 || days[1].Target != nil {
		t.Errorf("days = %+v", days)
	}
	if len(days[1].Workouts) != 0 || len(days[2].Workouts) != 1 {
		t.Errorf("the workout belongs to its local day: %+v", days[1:3])
	}

	// With a profile and weights: a day before the first weight takes the first one.
	repo.profile = &Profile{HeightCm: 180, BirthDate: date("1990-11-03"), Sex: Male, ActivityFactor: 1.3}
	repo.weights = []Weight{{MeasuredOn: date("2026-10-30"), WeightKg: 81}, {MeasuredOn: date("2026-10-28"), WeightKg: 80}}
	c.energy = map[string]float64{"2026-10-29": 12552}
	days, _ = s.Days(context.Background(), userID, "2026-10-26", "2026-11-01", "Europe/Zurich")
	if days[0].Target == nil || days[0].Target.WeightKg != 80 || days[6].Target.WeightKg != 81 {
		t.Errorf("weights = %+v, %+v", days[0].Target, days[6].Target)
	}
	if days[3].Target.Source != SourceWhoop || days[3].Target.Kcal != 3000 || days[2].Target.Source != SourceEstimate {
		t.Errorf("sources = %+v, %+v", days[2].Target, days[3].Target)
	}
	// The training day (the 28th) gets more energy than the rest day before.
	if days[2].Target.Kcal <= days[0].Target.Kcal {
		t.Errorf("training day %v, rest day %v", days[2].Target.Kcal, days[0].Target.Kcal)
	}
	if !days[0].Date.Equal(date("2026-10-26")) {
		t.Errorf("date = %v", days[0].Date)
	}

	// The body fat measured last on or before a day takes the resting energy from the lean mass;
	// the days before it was measured keep Mifflin-St Jeor.
	before := days
	repo.weights[1].BodyFatPercent = ptr(15.0)
	days, _ = s.Days(context.Background(), userID, "2026-10-26", "2026-11-01", "Europe/Zurich")
	if days[0].Target.Kcal != before[0].Target.Kcal || days[6].Target.Kcal == before[6].Target.Kcal {
		t.Errorf("body fat: before %+v, %+v; after %+v, %+v", before[0].Target, before[6].Target, days[0].Target, days[6].Target)
	}

	for _, bad := range [][3]string{
		{"2026-10-26", "2026-11-01", "Mars/Olympus"},
		{"2026-10-26", "2026-11-01", ""},
		{"2026-11-01", "2026-10-26", "UTC"},
		{"2025-01-01", "2026-11-01", "UTC"},
		{"26.10.2026", "2026-11-01", "UTC"},
	} {
		_, err := s.Days(context.Background(), userID, bad[0], bad[1], bad[2])
		wantKind(t, err, errs.KindInvalid)
	}
}

func TestSaveMacros(t *testing.T) {
	s, repo, _ := newTestService()
	userID := uuid.New()
	m, err := s.SaveMacros(context.Background(), userID, "2026-11-02", MacrosInput{ProteinG: ptr(180.0), Notes: "  cheat day "})
	if err != nil || !m.Day.Equal(date("2026-11-02")) || m.Notes != "cheat day" || len(repo.saved) != 1 {
		t.Errorf("saved = %+v, %v", m, err)
	}
	// Tomorrow is today somewhere.
	if _, err := s.SaveMacros(context.Background(), userID, "2026-11-03", MacrosInput{FatG: ptr(70.0)}); err != nil {
		t.Errorf("tomorrow: %v", err)
	}
	for _, in := range []MacrosInput{{}, {CarbsG: ptr(-1.0)}, {ProteinG: ptr(2001.0)}} {
		_, err := s.SaveMacros(context.Background(), userID, "2026-11-02", in)
		wantKind(t, err, errs.KindInvalid)
	}
	_, err = s.SaveMacros(context.Background(), userID, "2026-11-05", MacrosInput{ProteinG: ptr(1.0)})
	wantKind(t, err, errs.KindInvalid)
	wantKind(t, s.DeleteMacros(context.Background(), userID, "2026-11-02"), errs.KindNotFound)
}

func TestSaveProfileAndWeight(t *testing.T) {
	s, repo, _ := newTestService()
	userID := uuid.New()
	if err := s.SaveProfile(context.Background(), userID, ProfileInput{HeightCm: 180, BirthDate: "1990-05-01", Sex: Male}); err != nil {
		t.Fatal(err)
	}
	if repo.profile.ActivityFactor != defaultActivityFactor {
		t.Errorf("activity factor = %v", repo.profile.ActivityFactor)
	}
	for _, in := range []ProfileInput{
		{HeightCm: 90, BirthDate: "1990-05-01", Sex: Male},
		{HeightCm: 180, BirthDate: "2020-05-01", Sex: Male},
		{HeightCm: 180, BirthDate: "1990-05-01", Sex: "other"},
		{HeightCm: 180, BirthDate: "1990-05-01", Sex: Female, ActivityFactor: 2.5},
	} {
		wantKind(t, s.SaveProfile(context.Background(), userID, in), errs.KindInvalid)
	}

	if _, err := s.SaveWeight(context.Background(), userID, WeightInput{MeasuredOn: "2026-11-02", WeightKg: 80.4}); err != nil {
		t.Error(err)
	}
	_, err := s.SaveWeight(context.Background(), userID, WeightInput{MeasuredOn: "2026-11-02", WeightKg: 10})
	wantKind(t, err, errs.KindInvalid)
	_, err = s.SaveWeight(context.Background(), userID, WeightInput{MeasuredOn: "2026-11-02", WeightKg: 80, BodyFatPercent: ptr(70.0)})
	wantKind(t, err, errs.KindInvalid)
	wantKind(t, s.DeleteWeight(context.Background(), userID, uuid.New()), errs.KindNotFound)
}
