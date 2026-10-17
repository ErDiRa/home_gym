package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// fakeTrainings implements TrainingService; methods it doesn't override panic via the nil
// embedded interface.
type fakeTrainings struct {
	TrainingService
	err   error
	input training.Input // what Create received
}

func (f *fakeTrainings) Get(_ context.Context, id uuid.UUID) (training.Training, error) {
	return training.Training{GetTrainingRow: trainingrepos.GetTrainingRow{ID: id, Name: "Strength"}, Units: []training.Unit{}}, f.err
}

func (f *fakeTrainings) Create(_ context.Context, in training.Input) (trainingrepos.CreateTrainingRow, error) {
	f.input = in
	return trainingrepos.CreateTrainingRow{ID: uuid.New(), Name: in.Name}, f.err
}

func serve(t *testing.T, fake *fakeTrainings, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := New(Services{Trainings: fake}, nil, false)
	r := chi.NewRouter()
	r.Get("/trainings/{id}", h.GetTraining)
	r.Post("/trainings", h.CreateTraining)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct{ Error string }
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Error
}

func TestTrainingHandlers(t *testing.T) {
	id := uuid.New()
	rec := serve(t, &fakeTrainings{}, http.MethodGet, "/trainings/"+id.String(), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Strength"`) {
		t.Errorf("GET = %d %s", rec.Code, rec.Body)
	}

	fake := &fakeTrainings{}
	rec = serve(t, fake, http.MethodPost, "/trainings", `{"name":"Strength","units":[{"name":"Day A"}]}`)
	if rec.Code != http.StatusCreated || fake.input.Name != "Strength" || fake.input.Units[0].Name != "Day A" {
		t.Errorf("POST = %d %s, service got %+v", rec.Code, rec.Body, fake.input)
	}
}

func TestTrainingHandlerErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		method     string
		path, body string
		wantStatus int
		wantMsg    string
	}{
		{"invalid id", nil, http.MethodGet, "/trainings/nope", "", http.StatusBadRequest, "invalid training id"},
		{"invalid JSON", nil, http.MethodPost, "/trainings", "{", http.StatusBadRequest, "invalid JSON body"},
		{"invalid input", service.Invalid("name is required"), http.MethodPost, "/trainings", "{}", http.StatusBadRequest, "name is required"},
		{"not found", service.NotFound("training not found", errors.New("no rows")), http.MethodGet, "/trainings/" + uuid.NewString(), "", http.StatusNotFound, "training not found"},
		{"conflict", service.Conflict("taken", nil), http.MethodPost, "/trainings", "{}", http.StatusConflict, "taken"},
		{"internal error is not leaked", errors.New("connection refused"), http.MethodGet, "/trainings/" + uuid.NewString(), "", http.StatusInternalServerError, "internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, &fakeTrainings{err: tt.err}, tt.method, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := errorBody(t, rec); got != tt.wantMsg {
				t.Errorf("error = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
