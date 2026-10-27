//go:build component

package tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	trackingclient "github.com/erdira/homegym/backend/modules/tracking/api/http/client"
	whoopclient "github.com/erdira/homegym/backend/modules/whoop/api/http/client"
)

// The service under test has no WHOOP app configured: WHOOP can't be connected, and a workout has
// no WHOOP data.
func TestWhoopWithoutApp(t *testing.T) {
	t.Parallel()
	u, other := newUser(t), newUser(t)
	ctx := t.Context()

	status, err := u.Whoop.WhoopStatusWithResponse(ctx)
	if err != nil || status.StatusCode() != http.StatusOK || status.JSON200.Configured || status.JSON200.Connected {
		t.Fatalf("status = %v %s", err, body(status.Body))
	}
	noRedirect := *u.HTTP
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := noRedirect.Get(apiURL() + "/whoop/connect")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("connect without a WHOOP app = %d, want 404", resp.StatusCode)
	}

	exercise, reps := createExercise(t, u, exerciseInput("Squat")), repsUnit(t, u)
	activeTraining(t, u, exercise, reps)
	started, _ := u.Tracking.StartWorkoutWithResponse(ctx, trackingclient.StartWorkoutRequest{WeekSessionId: week(t, u, nil).Planned[0].Id})
	workout := started.JSON201.Id

	data, err := u.Whoop.GetWorkoutWhoopWithResponse(ctx, workout)
	if err != nil || data.StatusCode() != http.StatusOK || string(data.Body) != "null\n" {
		t.Errorf("WHOOP data of a workout without = %v %d %q", err, data.StatusCode(), data.Body)
	}
	if resp, _ := u.Whoop.UnlinkWhoopWithResponse(ctx, workout); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("unlink without data = %d, want 404", resp.StatusCode())
	}
	// Linking needs the workout to be the user's (checked through tracking's contract) before WHOOP.
	if resp, _ := other.Whoop.LinkWhoopWithResponse(ctx, workout, linkBody(uuid.New())); resp.StatusCode() != http.StatusNotFound {
		t.Errorf("link another user's workout = %d, want 404", resp.StatusCode())
	}
	if resp, _ := u.Whoop.LinkWhoopWithResponse(ctx, workout, linkBody(uuid.New())); resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("link without a connection = %d %s, want 400", resp.StatusCode(), body(resp.Body))
	}
}

func linkBody(whoopID uuid.UUID) whoopclient.LinkWhoopRequest {
	return whoopclient.LinkWhoopRequest{WhoopWorkoutId: whoopID}
}
