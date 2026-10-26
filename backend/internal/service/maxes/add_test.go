package maxes

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

var (
	kg      = uuid.New()
	percent = uuid.New()
)

func newFake() *fakeQueries {
	one := 1.0
	return &fakeQueries{units: map[uuid.UUID]*float64{kg: &one, percent: nil}, sets: map[uuid.UUID]repos.UserOwnsSetParams{}}
}

func TestAdd(t *testing.T) {
	f := newFake()
	user, exercise := uuid.New(), uuid.New()
	got, err := (&Service{q: f}).Add(context.Background(), user, exercise, MaxInput{
		LoadValue: 102.5, LoadUnit: kg, AchievedOn: "2026-10-20", Notes: " belt ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.created) != 1 || f.created[0].UserID != user || f.created[0].ExerciseID != exercise {
		t.Fatalf("created %+v", f.created)
	}
	if got.LoadValue != 102.5 || got.Notes != "belt" || got.AchievedOn.Time.Format(time.DateOnly) != "2026-10-20" || got.WorkoutSetID != nil {
		t.Errorf("max = %+v", got)
	}
}

// A max saved from a logged single keeps the set, if it is the user's set of that exercise.
func TestAddFromSet(t *testing.T) {
	f := newFake()
	user, exercise, set := uuid.New(), uuid.New(), uuid.New()
	f.sets[set] = repos.UserOwnsSetParams{UserID: user, ExerciseID: exercise}
	svc := &Service{q: f}
	in := MaxInput{LoadValue: 100, LoadUnit: kg, AchievedOn: "2026-10-20", WorkoutSetID: &set}
	if _, err := svc.Add(context.Background(), user, exercise, in); err != nil {
		t.Fatal(err)
	}
	if f.created[0].WorkoutSetID == nil || *f.created[0].WorkoutSetID != set {
		t.Errorf("workout set = %v, want %v", f.created[0].WorkoutSetID, set)
	}

	// Another user's set, or the set of another exercise, is unknown.
	_, err := svc.Add(context.Background(), uuid.New(), exercise, in)
	wantError(t, err, service.KindInvalid, "unknown set")
	_, err = svc.Add(context.Background(), user, uuid.New(), in)
	wantError(t, err, service.KindInvalid, "unknown set")
}

func TestAddRejects(t *testing.T) {
	future := time.Now().AddDate(0, 0, 3).Format(time.DateOnly)
	tests := []struct {
		name string
		in   MaxInput
		msg  string
	}{
		{"zero load", MaxInput{LoadValue: 0, LoadUnit: kg, AchievedOn: "2026-10-20"}, "load must be between 0 and 99999"},
		{"too heavy", MaxInput{LoadValue: 100000, LoadUnit: kg, AchievedOn: "2026-10-20"}, "load must be between 0 and 99999"},
		{"no date", MaxInput{LoadValue: 100, LoadUnit: kg}, "achieved_on must be a date (YYYY-MM-DD)"},
		{"future", MaxInput{LoadValue: 100, LoadUnit: kg, AchievedOn: future}, "achieved_on must not be in the future"},
		{"percent", MaxInput{LoadValue: 100, LoadUnit: percent, AchievedOn: "2026-10-20"}, "a 1RM needs a weight unit such as kg or lb"},
		{"unknown unit", MaxInput{LoadValue: 100, LoadUnit: uuid.New(), AchievedOn: "2026-10-20"}, "unknown load unit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			_, err := (&Service{q: f}).Add(context.Background(), uuid.New(), uuid.New(), tt.in)
			wantError(t, err, service.KindInvalid, tt.msg)
			if len(f.created) != 0 {
				t.Error("max was stored")
			}
		})
	}
}

func TestAddUnknownExercise(t *testing.T) {
	// Another user's or an unknown exercise inserts no row.
	f := newFake()
	f.createErr = pgx.ErrNoRows
	_, err := (&Service{q: f}).Add(context.Background(), uuid.New(), uuid.New(), MaxInput{LoadValue: 100, LoadUnit: kg, AchievedOn: "2026-10-20"})
	wantError(t, err, service.KindNotFound, "exercise not found")
}

// Users only ever see their own maxes: the list asks for the signed-in user's.
func TestListIsTheUsers(t *testing.T) {
	f := newFake()
	user := uuid.New()
	if _, err := (&Service{q: f}).List(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if len(f.listed) != 1 || f.listed[0] != user {
		t.Errorf("listed the maxes of %v, want only %v", f.listed, user)
	}
}
