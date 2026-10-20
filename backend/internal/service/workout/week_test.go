package workout

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

var (
	monday = time.Date(2026, 10, 12, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	sunday = monday.AddDate(0, 0, 7)
	// The Monday as the date the client sends.
	mondayDate = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
)

func date(y int, m time.Month, d int) pgtype.Date {
	return pgtype.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Valid: true}
}

func TestWeekNumber(t *testing.T) {
	tests := []struct {
		name  string
		start pgtype.Date
		want  *int32 // nil: outside the training's weeks
	}{
		{"first week", date(2026, 10, 12), ptr[int32](1)},
		{"third week", date(2026, 9, 28), ptr[int32](3)},
		{"last week", date(2026, 9, 21), ptr[int32](4)},
		{"after the end", date(2026, 9, 14), nil},
		{"before the start", date(2026, 10, 19), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			id := uuid.New()
			f.trainings = []repos.ListActiveTrainingsRow{{ID: id, Name: "Strength", StartDate: tt.start, Weeks: 4}}
			f.plan = []repos.ListWeekPlanRow{{ID: uuid.New(), SessionID: uuid.New(), Name: "Session 1"}}
			f.done = []repos.ListWorkoutsRow{{ID: uuid.New(), SessionName: "Session 1"}}

			got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday)
			if err != nil {
				t.Fatal(err)
			}
			if got.Training == nil || got.Training.Weeks != 4 || len(got.Done) != 1 {
				t.Errorf("week = %+v", got)
			}
			if tt.want == nil {
				if got.WeekNumber != nil || len(got.Planned) != 0 || got.Planned == nil || len(f.planned) != 0 {
					t.Errorf("week number = %v, planned = %v, want none", got.WeekNumber, got.Planned)
				}
				return
			}
			if got.WeekNumber == nil || *got.WeekNumber != *tt.want {
				t.Fatalf("week number = %v, want %d", got.WeekNumber, *tt.want)
			}
			if len(got.Planned) != 1 || f.planned[0] != (repos.ListWeekPlanParams{TrainingID: id, Week: *tt.want}) {
				t.Errorf("planned = %+v for %+v", got.Planned, f.planned)
			}
		})
	}
}

func TestWeekPicksTraining(t *testing.T) {
	mobility, strength := uuid.New(), uuid.New()
	tests := []struct {
		name string
		pick uuid.UUID
		want uuid.UUID
	}{
		{"the chosen one", strength, strength},
		{"the first without a choice", uuid.Nil, mobility},
		{"the first when the chosen one isn't active", uuid.New(), mobility},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			f.trainings = []repos.ListActiveTrainingsRow{
				{ID: mobility, Name: "Mobility", StartDate: date(2026, 10, 12), Weeks: 8},
				{ID: strength, Name: "Strength", StartDate: date(2026, 10, 5), Weeks: 4},
			}
			got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), tt.pick, mondayDate, monday, sunday)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Trainings) != 2 || got.Training == nil || got.Training.ID != tt.want {
				t.Fatalf("week = %+v, want training %v", got, tt.want)
			}
			if f.planned[0].TrainingID != tt.want {
				t.Errorf("planned of %v, want %v", f.planned[0].TrainingID, tt.want)
			}
		})
	}
}

func TestWeekWithoutTraining(t *testing.T) {
	f, _, _ := newFake()
	got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	// Workouts done still show without an active training.
	if got.Training != nil || got.WeekNumber != nil || got.Planned == nil || got.Done == nil {
		t.Errorf("week = %+v", got)
	}
	if l := f.listed[0]; !l.FromTime.Time.Equal(monday) || !l.ToTime.Time.Equal(sunday) {
		t.Errorf("listed %v to %v, want %v to %v", l.FromTime.Time, l.ToTime.Time, monday, sunday)
	}
}

func TestWeekNeedsMonday(t *testing.T) {
	f, _, _ := newFake()
	for _, d := range []time.Time{{}, mondayDate.AddDate(0, 0, 1)} {
		_, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, d, monday, sunday)
		wantError(t, err, service.KindInvalid, "date is required (YYYY-MM-DD), a Monday")
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
