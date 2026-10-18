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
	userID   uuid.UUID
	from, to time.Time
	set      workout.SetInput
}

func (f *fakeWorkouts) Week(_ context.Context, userID uuid.UUID, from, to time.Time) (workout.Week, error) {
	f.userID, f.from, f.to = userID, from, to
	if from.IsZero() || to.IsZero() {
		return workout.Week{}, service.Invalid("from and to are required")
	}
	return workout.Week{Units: []workoutrepos.ListTrainingUnitsRow{}, Done: []workoutrepos.ListWorkoutsRow{}}, nil
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
	r.Put("/workouts/{id}/sets", h.LogSet)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestGetWeek(t *testing.T) {
	fake := &fakeWorkouts{}
	rec := serveWorkouts(fake, http.MethodGet, "/week?from=2026-10-12T00:00:00%2B02:00&to=2026-10-19T00:00:00%2B02:00", "")
	if rec.Code != http.StatusOK || fake.userID != testUser {
		t.Fatalf("GET = %d %s for %v", rec.Code, rec.Body, fake.userID)
	}
	if want := time.Date(2026, 10, 11, 22, 0, 0, 0, time.UTC); !fake.from.Equal(want) || !fake.to.Equal(want.AddDate(0, 0, 7)) {
		t.Errorf("week = %v to %v, want from %v", fake.from, fake.to, want)
	}

	// An unparseable bound reaches the service as zero, which rejects it.
	rec = serveWorkouts(&fakeWorkouts{}, http.MethodGet, "/week?from=monday&to=2026-10-19T00:00:00Z", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET with a bad from = %d, want 400", rec.Code)
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
