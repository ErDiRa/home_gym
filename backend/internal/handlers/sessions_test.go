package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"

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
	rec := serveAPI(Services{Trainings: fake}, http.MethodPost, "/sessions", `{"name":"Session 1","blocks":[]}`)
	if rec.Code != http.StatusCreated || fake.userID != testUser || fake.input.Name != "Session 1" || !strings.Contains(rec.Body.String(), `"blocks":[]`) {
		t.Errorf("POST = %d %s", rec.Code, rec.Body)
	}

	id := uuid.New()
	rec = serveAPI(Services{Trainings: fake}, http.MethodDelete, "/sessions/"+id.String(), "")
	if rec.Code != http.StatusNotFound || fake.id != id || errorBody(t, rec) != "session not found" {
		t.Errorf("DELETE = %d %s", rec.Code, rec.Body)
	}
}
