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

// A cycle planned week by week has many active trainings; only the ones running this week are
// offered. Trainings without a cycle always are.
func TestWeekOffersCycleTrainingsOfTheWeek(t *testing.T) {
	cycle := uuid.New()
	thisWeek, nextWeek, lastWeek, mobility := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f, _, _ := newFake()
	f.trainings = []repos.ListActiveTrainingsRow{
		{ID: mobility, Name: "Mobility", StartDate: date(2026, 1, 5), Weeks: 4}, // ended long ago, no cycle
		{ID: lastWeek, Name: "Week 1", StartDate: date(2026, 10, 5), Weeks: 1, CycleID: &cycle},
		{ID: thisWeek, Name: "Week 2", StartDate: date(2026, 10, 12), Weeks: 1, CycleID: &cycle},
		{ID: nextWeek, Name: "Week 3", StartDate: date(2026, 10, 19), Weeks: 1, CycleID: &cycle},
	}

	got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), thisWeek, mondayDate, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	var offered []uuid.UUID
	for _, tr := range got.Trainings {
		offered = append(offered, tr.ID)
	}
	if len(offered) != 2 || offered[0] != mobility || offered[1] != thisWeek {
		t.Errorf("offered %v, want Mobility and Week 2", got.Trainings)
	}
	if got.Training == nil || got.Training.ID != thisWeek || got.WeekNumber == nil || *got.WeekNumber != 1 {
		t.Errorf("chosen %+v, week %v", got.Training, got.WeekNumber)
	}

	// A training of the cycle that isn't offered can't be chosen: the first offered one is.
	got, err = (&Service{q: f}).Week(context.Background(), uuid.New(), nextWeek, mondayDate, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	if got.Training == nil || got.Training.ID != mobility {
		t.Errorf("chosen %+v, want the first offered", got.Training)
	}
}

// When no training would be left, all are kept, so the tracker can say when they start or ended.
func TestWeekKeepsAllWhenNoneRuns(t *testing.T) {
	cycle := uuid.New()
	f, _, _ := newFake()
	f.trainings = []repos.ListActiveTrainingsRow{
		{ID: uuid.New(), Name: "Week 1", StartDate: date(2026, 11, 2), Weeks: 1, CycleID: &cycle},
		{ID: uuid.New(), Name: "Week 2", StartDate: date(2026, 11, 9), Weeks: 1, CycleID: &cycle},
	}

	got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Trainings) != 2 || got.Training == nil || got.WeekNumber != nil {
		t.Errorf("week = %+v", got)
	}
}

func TestWeekCycle(t *testing.T) {
	tests := []struct {
		name  string
		start pgtype.Date
		want  *int32 // nil: outside the cycle's weeks
	}{
		{"first week", date(2026, 10, 12), ptr[int32](1)},
		{"last week", date(2026, 8, 24), ptr[int32](8)},
		{"after the end", date(2026, 8, 17), nil},
		{"before the start", date(2026, 10, 19), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycleID := uuid.New()
			f, _, _ := newFake()
			// The training runs this week; the cycle may not (a training may lie outside it).
			f.trainings = []repos.ListActiveTrainingsRow{{ID: uuid.New(), Name: "Hips", StartDate: date(2026, 10, 12), Weeks: 1, CycleID: &cycleID}}
			f.cycle = &repos.GetWeekCycleRow{Name: "Mobility", StartDate: tt.start, Weeks: 8}
			f.goals = []repos.ListWeekCycleGoalsRow{{Title: "Better hips", Status: "open"}}

			got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday)
			if err != nil {
				t.Fatal(err)
			}
			c := got.Cycle
			if c == nil || c.ID != cycleID || c.Name != "Mobility" || c.Weeks != 8 || len(c.Goals) != 1 {
				t.Fatalf("cycle = %+v", c)
			}
			if (tt.want == nil) != (c.WeekNumber == nil) || (tt.want != nil && *c.WeekNumber != *tt.want) {
				t.Errorf("cycle week = %v, want %v", c.WeekNumber, tt.want)
			}
		})
	}
}

func TestWeekWithoutCycle(t *testing.T) {
	f, _, _ := newFake()
	f.trainings = []repos.ListActiveTrainingsRow{{ID: uuid.New(), Name: "Strength", StartDate: date(2026, 10, 12), Weeks: 4}}
	got, err := (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cycle != nil {
		t.Errorf("cycle = %+v, want none", got.Cycle)
	}
	// A cycle deleted meanwhile is none either.
	cycleID := uuid.New()
	f.trainings[0].CycleID = &cycleID
	if got, err = (&Service{q: f}).Week(context.Background(), uuid.New(), uuid.Nil, mondayDate, monday, sunday); err != nil || got.Cycle != nil {
		t.Errorf("cycle = %+v, err %v", got.Cycle, err)
	}
}
