package handlers

import (
	"context"
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

// fakeLibrary implements the library part of TrainingService.
type fakeLibrary struct {
	TrainingService
	userID, id uuid.UUID
	input      training.SessionInput
}

func (f *fakeLibrary) CreateSession(_ context.Context, userID uuid.UUID, in training.SessionInput) (training.LibrarySession, error) {
	f.userID, f.input = userID, in
	return training.LibrarySession{GetLibrarySessionRow: trainingrepos.GetLibrarySessionRow{ID: uuid.New(), Name: in.Name}, Blocks: []training.Block{}}, nil
}

func (f *fakeLibrary) DeleteSession(_ context.Context, userID, id uuid.UUID) error {
	f.userID, f.id = userID, id
	return service.NotFound("session not found", nil)
}

func TestSessionHandlers(t *testing.T) {
	fake := &fakeLibrary{}
	h := New(Services{Trainings: fake}, nil, false)
	r := chi.NewRouter()
	r.Use(signedIn(testUser))
	r.Post("/sessions", h.CreateSession)
	r.Delete("/sessions/{id}", h.DeleteSession)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"Session 1","blocks":[]}`)))
	if rec.Code != http.StatusCreated || fake.userID != testUser || fake.input.Name != "Session 1" || !strings.Contains(rec.Body.String(), `"blocks":[]`) {
		t.Errorf("POST = %d %s", rec.Code, rec.Body)
	}

	id := uuid.New()
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/"+id.String(), nil))
	if rec.Code != http.StatusNotFound || fake.id != id || errorBody(t, rec) != "session not found" {
		t.Errorf("DELETE = %d %s", rec.Code, rec.Body)
	}
}
