package workout

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

var (
	monday = time.Date(2026, 10, 12, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	sunday = monday.AddDate(0, 0, 7)
)

func TestWeekRotatesUnits(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name string
		last *uuid.UUID
		want uuid.UUID
	}{
		{"nothing done yet", nil, a},
		{"after A comes B", &a, b},
		{"after the last unit comes the first", &c, a},
		{"the last unit was removed", ptr(uuid.New()), a},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			f.training = &repos.GetActiveTrainingRow{ID: uuid.New(), Name: "Strength", SessionsPerWeek: 3}
			f.units = []repos.ListTrainingUnitsRow{{ID: a, Name: "A"}, {ID: b, Name: "B"}, {ID: c, Name: "C"}}
			f.lastUnit = tt.last
			f.done = []repos.ListWorkoutsRow{{ID: uuid.New(), UnitName: "A"}}

			got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), monday, sunday)
			if err != nil {
				t.Fatal(err)
			}
			if got.Training == nil || got.Training.SessionsPerWeek != 3 || len(got.Units) != 3 || len(got.Done) != 1 {
				t.Errorf("week = %+v", got)
			}
			if got.NextUnitID == nil || *got.NextUnitID != tt.want {
				t.Errorf("next unit = %v, want %v", got.NextUnitID, tt.want)
			}
		})
	}
}

func TestWeekWithoutTraining(t *testing.T) {
	f, _, _ := newFake()
	current := repos.Workout{ID: uuid.New(), UnitName: "Day A"}
	f.current = &current

	got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	// Workouts done and the one in progress still show without an active training.
	if got.Training != nil || got.NextUnitID != nil || got.Units == nil || got.Done == nil || got.Current == nil || got.Current.ID != current.ID {
		t.Errorf("week = %+v", got)
	}
	if l := f.listed[0]; !l.FromTime.Time.Equal(monday) || !l.ToTime.Time.Equal(sunday) {
		t.Errorf("listed %v to %v, want %v to %v", l.FromTime.Time, l.ToTime.Time, monday, sunday)
	}
}

func TestListPeriod(t *testing.T) {
	f, _, _ := newFake()
	s := &Service{q: f}
	for _, p := range []struct{ from, to time.Time }{
		{time.Time{}, sunday},
		{sunday, monday},
		{monday, monday},
	} {
		_, err := s.List(context.Background(), uuid.New(), p.from, p.to)
		wantError(t, err, service.KindInvalid, "from and to are required (RFC 3339), from before to")
	}
	_, err := s.List(context.Background(), uuid.New(), monday, monday.AddDate(1, 0, 2))
	wantError(t, err, service.KindInvalid, "the period can be at most a year")
}

func TestCurrentNone(t *testing.T) {
	f, _, _ := newFake()
	got, err := (&Service{q: f}).Current(context.Background(), uuid.New())
	if err != nil || got != nil {
		t.Errorf("current = %+v, %v", got, err)
	}
}
