package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle"
	cyclerepos "github.com/erdira/homegym/backend/internal/service/cycle/repos"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// fakeCycles implements CycleService; methods it doesn't override panic via the nil embedded
// interface.
type fakeCycles struct {
	CycleService
	err           error
	userID        uuid.UUID   // whose cycle was asked for
	input         cycle.Input // what Create/Update received
	withTrainings *bool       // what Delete received
}

func (f *fakeCycles) Get(_ context.Context, userID, id uuid.UUID) (cycle.Cycle, error) {
	f.userID = userID
	return cycle.Cycle{GetCycleRow: cyclerepos.GetCycleRow{ID: id, Name: "Mobility"},
		Goals: []cyclerepos.ListCycleGoalsRow{}, Trainings: []cyclerepos.ListCycleTrainingsRow{}}, f.err
}

func (f *fakeCycles) Create(_ context.Context, userID uuid.UUID, in cycle.Input) (cycle.Cycle, error) {
	f.userID, f.input = userID, in
	return cycle.Cycle{GetCycleRow: cyclerepos.GetCycleRow{ID: uuid.New(), Name: in.Name}}, f.err
}

func (f *fakeCycles) Update(_ context.Context, userID, id uuid.UUID, in cycle.Input) (cycle.Cycle, error) {
	f.userID, f.input = userID, in
	return cycle.Cycle{GetCycleRow: cyclerepos.GetCycleRow{ID: id, Name: in.Name}}, f.err
}

func (f *fakeCycles) Delete(_ context.Context, userID, _ uuid.UUID, withTrainings bool) error {
	f.userID, f.withTrainings = userID, &withTrainings
	return f.err
}

func serveCycles(t *testing.T, fake *fakeCycles, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serveAPI(Services{Cycles: fake}, method, path, body)
}

func TestCycleHandlers(t *testing.T) {
	id := uuid.NewString()
	fake := &fakeCycles{}
	rec := serveCycles(t, fake, http.MethodGet, "/cycles/"+id, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Mobility"`) ||
		!strings.Contains(rec.Body.String(), `"goals":[]`) || fake.userID != testUser {
		t.Errorf("GET = %d %s for %v", rec.Code, rec.Body, fake.userID)
	}

	fake = &fakeCycles{}
	rec = serveCycles(t, fake, http.MethodPost, "/cycles",
		`{"name":"Mobility","start_date":"2026-03-02","weeks":8,"goals":[{"title":"Better hips","target":"2 min"}]}`)
	if rec.Code != http.StatusCreated || fake.input.Name != "Mobility" || fake.input.Weeks != 8 ||
		len(fake.input.Goals) != 1 || fake.input.Goals[0].Target != "2 min" || fake.userID != testUser {
		t.Errorf("POST = %d %s, service got %+v", rec.Code, rec.Body, fake.input)
	}

	fake = &fakeCycles{}
	rec = serveCycles(t, fake, http.MethodPut, "/cycles/"+id, `{"name":"Hips"}`)
	if rec.Code != http.StatusOK || fake.input.Name != "Hips" {
		t.Errorf("PUT = %d %s", rec.Code, rec.Body)
	}
}

func TestDeleteCycleTrainingsParam(t *testing.T) {
	for query, want := range map[string]bool{"": false, "?trainings=keep": false, "?trainings=delete": true} {
		fake := &fakeCycles{}
		rec := serveCycles(t, fake, http.MethodDelete, "/cycles/"+uuid.NewString()+query, "")
		if rec.Code != http.StatusNoContent || fake.withTrainings == nil || *fake.withTrainings != want {
			t.Errorf("DELETE%s = %d, withTrainings %v, want %v", query, rec.Code, fake.withTrainings, want)
		}
	}

	fake := &fakeCycles{}
	rec := serveCycles(t, fake, http.MethodDelete, "/cycles/"+uuid.NewString()+"?trainings=all", "")
	if rec.Code != http.StatusBadRequest || errorBody(t, rec) != "trainings must be keep or delete" || fake.withTrainings != nil {
		t.Errorf("DELETE ?trainings=all = %d", rec.Code)
	}
}

func TestCycleHandlerErrors(t *testing.T) {
	notFound := service.NotFound("cycle not found", nil)
	tests := []struct {
		name       string
		err        error
		method     string
		path, body string
		wantStatus int
		wantMsg    string
	}{
		{"invalid id", nil, http.MethodGet, "/cycles/nope", "", http.StatusBadRequest, "invalid id"},
		{"invalid JSON", nil, http.MethodPost, "/cycles", "{", http.StatusBadRequest, "invalid JSON body"},
		{"invalid input", service.Invalid("start_date must be a Monday"), http.MethodPost, "/cycles", "{}", http.StatusBadRequest, "start_date must be a Monday"},
		{"another user's cycle", notFound, http.MethodGet, "/cycles/" + uuid.NewString(), "", http.StatusNotFound, "cycle not found"},
		{"update another user's", notFound, http.MethodPut, "/cycles/" + uuid.NewString(), "{}", http.StatusNotFound, "cycle not found"},
		{"delete another user's", notFound, http.MethodDelete, "/cycles/" + uuid.NewString(), "", http.StatusNotFound, "cycle not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveCycles(t, &fakeCycles{err: tt.err}, tt.method, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := errorBody(t, rec); got != tt.wantMsg {
				t.Errorf("error = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func (f *fakeTrainings) ImportCycle(_ context.Context, userID uuid.UUID, in training.CycleImportInput) (training.ImportedCycle, error) {
	f.userID = userID
	return training.ImportedCycle{CreateCycleRow: trainingrepos.CreateCycleRow{ID: uuid.New(), Name: in.Cycle.Name}}, f.err
}

// A cycle import may be larger than other bodies (1 MiB), up to 8 MiB.
func TestImportCycleBodyLimit(t *testing.T) {
	post := func(path, body string) *httptest.ResponseRecorder {
		return serveAPI(Services{Trainings: &fakeTrainings{}, Cycles: &fakeCycles{}}, http.MethodPost, path, body)
	}
	name := func(size int) string { return `{"name":"` + strings.Repeat("x", size) + `"}` }

	if rec := post("/cycles/import", `{"cycle":`+name(2<<20)+`}`); rec.Code != http.StatusCreated {
		t.Errorf("import of 2 MiB = %d %s", rec.Code, rec.Body)
	}
	if rec := post("/cycles", name(2<<20)); rec.Code != http.StatusBadRequest {
		t.Errorf("create of 2 MiB = %d, want the 1 MiB limit", rec.Code)
	}
	if rec := post("/cycles/import", `{"cycle":`+name(9<<20)+`}`); rec.Code != http.StatusBadRequest {
		t.Errorf("import of 9 MiB = %d, want 400", rec.Code)
	}
}
