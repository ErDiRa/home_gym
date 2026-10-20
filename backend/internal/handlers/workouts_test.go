package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout"
	workoutrepos "github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// fakeWorkouts implements WorkoutService; methods it doesn't override panic via the nil
// embedded interface.
type fakeWorkouts struct {
	WorkoutService
	userID, training uuid.UUID
	monday, from, to time.Time
	set              workout.SetInput
	slot             uuid.UUID
	inProgress       bool // Start finds the slot already in progress
}

func (f *fakeWorkouts) Week(_ context.Context, userID, trainingID uuid.UUID, monday, from, to time.Time) (workout.Week, error) {
	f.userID, f.training, f.monday, f.from, f.to = userID, trainingID, monday, from, to
	if monday.IsZero() || from.IsZero() || to.IsZero() {
		return workout.Week{}, service.Invalid("date, from and to are required")
	}
	return workout.Week{Planned: []workoutrepos.ListWeekPlanRow{}, Done: []workoutrepos.ListWorkoutsRow{}}, nil
}

func (f *fakeWorkouts) Start(_ context.Context, userID, weekSessionID uuid.UUID) (workout.Workout, bool, error) {
	f.userID, f.slot = userID, weekSessionID
	return workout.Workout{ID: uuid.New(), WeekSessionID: &weekSessionID}, !f.inProgress, nil
}

func (f *fakeWorkouts) LogSet(_ context.Context, userID, workoutID uuid.UUID, in workout.SetInput) (workoutrepos.WorkoutSet, error) {
	f.userID, f.set = userID, in
	return workoutrepos.WorkoutSet{ID: uuid.New(), WorkoutID: workoutID}, nil
}

func serveWorkouts(fake *fakeWorkouts, method, path, body string) *httptest.ResponseRecorder {
	h := New(Services{Workouts: fake}, nil, false)
	r := chi.NewRouter()
	r.Use(signedIn(testUser))
	r.Get("/week", h.GetWeek)
	r.Post("/workouts", h.StartWorkout)
	r.Put("/workouts/{id}/sets", h.LogSet)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestGetWeek(t *testing.T) {
	fake := &fakeWorkouts{}
	rec := serveWorkouts(fake, http.MethodGet, "/week?date=2026-10-12&from=2026-10-12T00:00:00%2B02:00&to=2026-10-19T00:00:00%2B02:00", "")
	if rec.Code != http.StatusOK || fake.userID != testUser {
		t.Fatalf("GET = %d %s for %v", rec.Code, rec.Body, fake.userID)
	}
	if want := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC); !fake.monday.Equal(want) || fake.training != uuid.Nil {
		t.Errorf("monday = %v, training = %v, want %v and none", fake.monday, fake.training, want)
	}
	training, picked := uuid.New(), &fakeWorkouts{}
	serveWorkouts(picked, http.MethodGet, "/week?training="+training.String()+"&date=2026-10-12&from=2026-10-12T00:00:00Z&to=2026-10-19T00:00:00Z", "")
	if picked.training != training {
		t.Errorf("training = %v, want %v", picked.training, training)
	}
	if want := time.Date(2026, 10, 11, 22, 0, 0, 0, time.UTC); !fake.from.Equal(want) || !fake.to.Equal(want.AddDate(0, 0, 7)) {
		t.Errorf("week = %v to %v, want from %v", fake.from, fake.to, want)
	}

	// An unparseable bound reaches the service as zero, which rejects it.
	rec = serveWorkouts(&fakeWorkouts{}, http.MethodGet, "/week?date=2026-10-12&from=monday&to=2026-10-19T00:00:00Z", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET with a bad from = %d, want 400", rec.Code)
	}
	rec = serveWorkouts(&fakeWorkouts{}, http.MethodGet, "/week?date=12.10.2026&from=2026-10-12T00:00:00Z&to=2026-10-19T00:00:00Z", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET with a bad date = %d, want 400", rec.Code)
	}
}

func TestStartWorkout(t *testing.T) {
	fake := &fakeWorkouts{}
	slot := uuid.New()
	rec := serveWorkouts(fake, http.MethodPost, "/workouts", `{"week_session_id":"`+slot.String()+`"}`)
	if rec.Code != http.StatusCreated || fake.userID != testUser || fake.slot != slot {
		t.Errorf("POST = %d %s, service got %v", rec.Code, rec.Body, fake.slot)
	}
	// The slot's workout in progress comes back as is.
	rec = serveWorkouts(&fakeWorkouts{inProgress: true}, http.MethodPost, "/workouts", `{"week_session_id":"`+slot.String()+`"}`)
	if rec.Code != http.StatusOK {
		t.Errorf("POST of a slot in progress = %d, want 200", rec.Code)
	}
}

func TestLogSet(t *testing.T) {
	fake := &fakeWorkouts{}
	exercise := uuid.New()
	rec := serveWorkouts(fake, http.MethodPut, "/workouts/"+uuid.NewString()+"/sets",
		`{"exercise_id":"`+exercise.String()+`","block_position":1,"exercise_position":2,"set_number":3,"reps_value":8}`)
	if rec.Code != http.StatusOK || fake.userID != testUser || fake.set.ExerciseID != exercise || fake.set.SetNumber != 3 || *fake.set.RepsValue != 8 {
		t.Errorf("PUT = %d %s, service got %+v", rec.Code, rec.Body, fake.set)
	}

	rec = serveWorkouts(&fakeWorkouts{}, http.MethodPut, "/workouts/nope/sets", `{}`)
	if rec.Code != http.StatusBadRequest || errorBody(t, rec) != "invalid workout id" {
		t.Errorf("PUT with a bad id = %d", rec.Code)
	}
}
