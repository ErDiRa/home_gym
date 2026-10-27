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

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// fakeTrainings implements TrainingService; methods it doesn't override panic via the nil
// embedded interface.
type fakeTrainings struct {
	TrainingService
	err    error
	userID uuid.UUID      // whose training was asked for
	input  training.Input // what Create received
}

func (f *fakeTrainings) Get(_ context.Context, userID, id uuid.UUID) (training.Training, error) {
	f.userID = userID
	return training.Training{GetTrainingRow: trainingrepos.GetTrainingRow{ID: id, Name: "Strength"}, Sessions: []training.Session{}, Schedule: [][]training.Slot{}}, f.err
}

func (f *fakeTrainings) Create(_ context.Context, userID uuid.UUID, in training.Input) (trainingrepos.CreateTrainingRow, error) {
	f.userID, f.input = userID, in
	return trainingrepos.CreateTrainingRow{ID: uuid.New(), Name: in.Name}, f.err
}

// signedIn puts userID into every request's context the way auth.Middleware does.
func signedIn(userID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &auth.Claims{}
			claims.Subject = userID.String()
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

// testUser is the signed-in user of the handler tests.
var testUser = uuid.New()

// serveAPI serves one request to /api<path> through the generated routes, with testUser
// signed in.
func serveAPI(s Services, method, path, body string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	New(s, nil, false).routes(r, signedIn(testUser))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, "/api"+path, strings.NewReader(body)))
	return rec
}

func serve(t *testing.T, fake *fakeTrainings, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serveAPI(Services{Trainings: fake}, method, path, body)
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
	fake := &fakeTrainings{}
	rec := serve(t, fake, http.MethodGet, "/trainings/"+id.String(), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Strength"`) || fake.userID != testUser {
		t.Errorf("GET = %d %s for %v", rec.Code, rec.Body, fake.userID)
	}

	fake = &fakeTrainings{}
	rec = serve(t, fake, http.MethodPost, "/trainings", `{"name":"Strength","weeks":4,"sessions":[{"name":"Session 1"}],"schedule":[[{"session":0}],[],[],[]]}`)
	if rec.Code != http.StatusCreated || fake.input.Name != "Strength" || fake.input.Weeks != 4 || fake.input.Sessions[0].Name != "Session 1" ||
		len(fake.input.Schedule) != 4 || fake.input.Schedule[0][0].Session != 0 || fake.userID != testUser {
		t.Errorf("POST = %d %s, service got %+v for %v", rec.Code, rec.Body, fake.input, fake.userID)
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
		{"invalid id", nil, http.MethodGet, "/trainings/nope", "", http.StatusBadRequest, "invalid id"},
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
