//go:build component

package tests

import (
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	catalogclient "github.com/erdira/homegym/backend/modules/catalog/api/http/client"
)

// joints returns the shared joints by name.
func joints(t *testing.T, u user) map[string]uuid.UUID {
	t.Helper()
	resp, err := u.Catalog.ListJointsWithResponse(t.Context())
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("list joints: %v %s", err, body(resp.Body))
	}
	out := map[string]uuid.UUID{}
	for _, j := range *resp.JSON200 {
		out[j.Name] = j.Id
	}
	return out
}

// createExercise adds an exercise to u's library and returns its id.
func createExercise(t *testing.T, u user, in catalogclient.ExerciseInput) uuid.UUID {
	t.Helper()
	resp, err := u.Catalog.CreateExerciseWithResponse(t.Context(), in)
	if err != nil || resp.StatusCode() != http.StatusCreated {
		t.Fatalf("create exercise: %v %s", err, body(resp.Body))
	}
	return resp.JSON201.Id
}

func exerciseInput(name string, jointIDs ...uuid.UUID) catalogclient.ExerciseInput {
	return catalogclient.ExerciseInput{
		Name: name, Type: catalogclient.Strength, JointIds: append([]uuid.UUID{}, jointIDs...),
		MuscleGroupIds: []uuid.UUID{}, FocusIds: []uuid.UUID{}, VideoUrls: []string{},
	}
}

func TestExerciseLibrary(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	j := joints(t, u)

	in := exerciseInput("Squat", j["Knee"], j["Hip"])
	in.VideoUrls = []string{" https://www.youtube.com/watch?v=ultWZbUMPL8 ", ""}
	id := createExercise(t, u, in)

	// The list shows the links by name and the trimmed video.
	list, err := u.Catalog.ListExercisesWithResponse(ctx)
	if err != nil || list.StatusCode() != http.StatusOK || len(*list.JSON200) != 1 {
		t.Fatalf("list = %v %s", err, body(list.Body))
	}
	got := (*list.JSON200)[0]
	if got.Id != id || !slices.Equal(got.Joints, []string{"Hip", "Knee"}) ||
		!slices.Equal(got.VideoUrls, []string{"https://www.youtube.com/watch?v=ultWZbUMPL8"}) || len(got.Focuses) != 0 {
		t.Errorf("listed = %+v", got)
	}

	// Updating replaces the links; the form gets them as ids.
	in = exerciseInput("Back Squat", j["Knee"])
	in.Type = catalogclient.Mobility
	if resp, err := u.Catalog.UpdateExerciseWithResponse(ctx, id, in); err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("update = %v %s", err, body(resp.Body))
	}
	detail, err := u.Catalog.GetExerciseWithResponse(ctx, id)
	if err != nil || detail.StatusCode() != http.StatusOK {
		t.Fatalf("get = %v %s", err, body(detail.Body))
	}
	if d := detail.JSON200; d.Name != "Back Squat" || d.Type != catalogclient.Mobility || !slices.Equal(d.JointIds, []uuid.UUID{j["Knee"]}) {
		t.Errorf("after update = %+v", d)
	}
}

func TestExercisesAreTheUsersOwn(t *testing.T) {
	t.Parallel()
	owner, other := newUser(t), newUser(t)
	ctx := t.Context()
	id := createExercise(t, owner, exerciseInput("Deadlift"))

	if resp, _ := other.Catalog.GetExerciseWithResponse(ctx, id); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("another user's get = %d, want 404", resp.StatusCode())
	}
	if resp, _ := other.Catalog.UpdateExerciseWithResponse(ctx, id, exerciseInput("Mine now")); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("another user's update = %d, want 404", resp.StatusCode())
	}
	if resp, _ := other.Catalog.RateExerciseWithResponse(ctx, id, catalogclient.RatingInput{Rating: 5}); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("another user's rating = %d, want 404", resp.StatusCode())
	}
	if list, _ := other.Catalog.ListExercisesWithResponse(ctx); len(*list.JSON200) != 0 {
		t.Errorf("another user lists %d exercises, want none", len(*list.JSON200))
	}
}

func TestExerciseValidation(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	tests := []struct {
		name    string
		in      catalogclient.ExerciseInput
		wantMsg string
	}{
		{"no name", exerciseInput(""), "name is required"},
		{"unknown joint", exerciseInput("Squat", uuid.New()), "unknown joint, muscle group or focus"},
		{"bad video", func() catalogclient.ExerciseInput {
			in := exerciseInput("Squat")
			in.VideoUrls = []string{"ftp://example.com"}
			return in
		}(), "video links must be http(s) URLs (max 500 characters)"},
	}
	for _, tt := range tests {
		resp, err := u.Catalog.CreateExerciseWithResponse(t.Context(), tt.in)
		if err != nil || resp.StatusCode() != http.StatusBadRequest || resp.JSON400.Error != tt.wantMsg {
			t.Errorf("%s: %v %d %s, want 400 %q", tt.name, err, resp.StatusCode(), body(resp.Body), tt.wantMsg)
		}
	}
}

func TestRatings(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	id := createExercise(t, u, exerciseInput("Pull-up"))

	if resp, _ := u.Catalog.RateExerciseWithResponse(ctx, id, catalogclient.RatingInput{Rating: 6}); resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("rating 6 = %d, want 400", resp.StatusCode())
	}
	for _, rating := range []int32{3, 5} { // rating again replaces the first
		resp, err := u.Catalog.RateExerciseWithResponse(ctx, id, catalogclient.RatingInput{Rating: rating, Notes: " shoulders "})
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Fatalf("rate %d = %v %s", rating, err, body(resp.Body))
		}
	}
	list, err := u.Catalog.ListExerciseRatingsWithResponse(ctx)
	if err != nil || len(*list.JSON200) != 1 {
		t.Fatalf("ratings = %v %s", err, body(list.Body))
	}
	if r := (*list.JSON200)[0]; r.ExerciseId != id || r.Rating != 5 || r.Notes != "shoulders" {
		t.Errorf("rating = %+v", r)
	}
}

// Focuses and units are shared by all users; their names are unique.
func TestSharedLists(t *testing.T) {
	t.Parallel()
	u := newUser(t)
	ctx := t.Context()
	name := "Focus " + uuid.NewString()[:8]

	focus, err := u.Catalog.CreateFocusWithResponse(ctx, catalogclient.FocusInput{Name: name, Alias: "T"})
	if err != nil || focus.StatusCode() != http.StatusCreated {
		t.Fatalf("create focus = %v %s", err, body(focus.Body))
	}
	if dup, _ := u.Catalog.CreateFocusWithResponse(ctx, catalogclient.FocusInput{Name: name, Alias: "T"}); dup.StatusCode() != http.StatusConflict {
		t.Errorf("duplicate focus = %d, want 409", dup.StatusCode())
	}
	focuses, _ := newUser(t).Catalog.ListFocusesWithResponse(ctx)
	if !slices.ContainsFunc(*focuses.JSON200, func(f catalogclient.Focus) bool { return f.Id == focus.JSON201.Id }) {
		t.Error("another user doesn't see the new focus")
	}

	unitName := "u" + uuid.NewString()[:8]
	maxValue := 10.0
	unit, err := u.Catalog.CreateLoadUnitWithResponse(ctx, catalogclient.UnitInput{Name: unitName, MaxValue: &maxValue})
	if err != nil || unit.StatusCode() != http.StatusCreated || unit.JSON201.KgPerUnit != nil || unit.JSON201.PercentOfMax {
		t.Fatalf("create load unit = %v %s", err, body(unit.Body))
	}
	units, _ := u.Catalog.ListLoadUnitsWithResponse(ctx)
	if last := (*units.JSON200)[len(*units.JSON200)-1]; last.Id != unit.JSON201.Id || *last.MaxValue != 10 {
		t.Errorf("new units come last; last = %+v", last)
	}

	kinds, err := u.Catalog.ListBlockKindsWithResponse(ctx)
	if err != nil || len(*kinds.JSON200) != 4 || (*kinds.JSON200)[3].Code != "wod" || !(*kinds.JSON200)[3].Scored {
		t.Errorf("block kinds = %v %s", err, body(kinds.Body))
	}
}
