//go:build component

package tests

import (
	"net/http"
	"testing"
	"time"

	nutritionclient "github.com/erdira/homegym/backend/modules/nutrition/api/http/client"
	trackingclient "github.com/erdira/homegym/backend/modules/tracking/api/http/client"
)

// A day shows the macros logged, the target estimated from the body (WHOOP isn't connected) and
// the workouts finished that day (read through tracking's contract); another user sees none of it.
func TestNutritionDays(t *testing.T) {
	t.Parallel()
	u, other := newUser(t), newUser(t)
	ctx := t.Context()
	today := time.Now().UTC().Format(time.DateOnly)
	days := func(u user) []nutritionclient.NutritionDay {
		t.Helper()
		resp, err := u.Nutrition.ListNutritionDaysWithResponse(ctx, &nutritionclient.ListNutritionDaysParams{From: today, To: today, Tz: "UTC"})
		if err != nil || resp.StatusCode() != http.StatusOK || len(*resp.JSON200) != 1 {
			t.Fatalf("days = %v %s", err, body(resp.Body))
		}
		return *resp.JSON200
	}

	// Without a body there is no target yet.
	if d := days(u)[0]; d.Target != nil || d.Macros != nil || len(d.Workouts) != 0 {
		t.Errorf("empty day = %+v", d)
	}
	profile, _ := u.Nutrition.GetNutritionProfileWithResponse(ctx)
	if string(profile.Body) != "null\n" {
		t.Errorf("profile before = %q", profile.Body)
	}
	saved, err := u.Nutrition.SaveNutritionProfileWithResponse(ctx, nutritionclient.ProfileInput{HeightCm: 180, BirthDate: "1990-05-01", Sex: nutritionclient.Male})
	if err != nil || saved.StatusCode() != http.StatusOK || saved.JSON200.ActivityFactor != 1.3 {
		t.Fatalf("save profile = %v %s", err, body(saved.Body))
	}
	if resp, _ := u.Nutrition.SaveBodyWeightWithResponse(ctx, nutritionclient.BodyWeightInput{MeasuredOn: today, WeightKg: 80}); resp.StatusCode() != http.StatusOK {
		t.Fatalf("save weight = %s", body(resp.Body))
	}
	rest := days(u)[0].Target
	if rest == nil || rest.Source != nutritionclient.Estimate || rest.ProteinG != 160 || rest.WeightKg != 80 {
		t.Fatalf("target = %+v", rest)
	}

	protein := 170.0
	logged, err := u.Nutrition.SaveDayMacrosWithResponse(ctx, today, nutritionclient.MacrosInput{ProteinG: &protein})
	if err != nil || logged.StatusCode() != http.StatusOK || logged.JSON200.Kcal != 680 {
		t.Fatalf("save macros = %v %s", err, body(logged.Body))
	}
	if resp, _ := u.Nutrition.SaveDayMacrosWithResponse(ctx, today, nutritionclient.MacrosInput{}); resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("no macros = %d, want 400", resp.StatusCode())
	}

	// A finished workout belongs to its day and adds to the energy.
	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	activeTraining(t, u, exercise, reps)
	started, _ := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: week(t, u, nil).Planned[0].Id})
	if resp, _ := u.Tracking.FinishWorkoutWithResponse(ctx, started.JSON201.Id, trackingclient.FinishWorkoutRequest{}); resp.StatusCode() != http.StatusOK {
		t.Fatalf("finish = %s", body(resp.Body))
	}
	d := days(u)[0]
	if d.Macros == nil || *d.Macros.ProteinG != 170 || len(d.Workouts) != 1 || d.Workouts[0].Id != started.JSON201.Id {
		t.Errorf("day = %+v", d)
	}
	if d.Target == nil || d.Target.Kcal < rest.Kcal {
		t.Errorf("training day target = %+v, rest day %+v", d.Target, rest)
	}

	if d := days(other)[0]; d.Macros != nil || d.Target != nil || len(d.Workouts) != 0 {
		t.Errorf("another user's day = %+v", d)
	}
	if resp, _ := other.Nutrition.DeleteDayMacrosWithResponse(ctx, today); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("delete another user's macros = %d, want 404", resp.StatusCode())
	}
	if resp, _ := u.Nutrition.DeleteDayMacrosWithResponse(ctx, today); resp.StatusCode() != http.StatusNoContent {
		t.Errorf("delete macros = %d", resp.StatusCode())
	}
	if resp, _ := u.Nutrition.ListNutritionDaysWithResponse(ctx, &nutritionclient.ListNutritionDaysParams{From: today, To: today, Tz: "Nowhere"}); resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("unknown time zone = %d, want 400", resp.StatusCode())
	}
}
