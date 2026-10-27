package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/logging"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/whoop"
	whooprepos "github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

// whoopStateCookie carries the OAuth state from ConnectWhoop to WhoopCallback, so the callback
// only accepts a code for the connection this browser started.
const whoopStateCookie = "whoop_state"

// settingsPage is where the browser returns to after connecting WHOOP.
const settingsPage = "/track/settings"

// WhoopStatus tells whether WHOOP can be connected (the app is configured) and whether it is.
func (h *Handler) WhoopStatus(ctx context.Context, _ api.WhoopStatusRequestObject) (api.WhoopStatusResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	connected, err := h.whoop.Connected(ctx, userID)
	if err != nil {
		return nil, err
	}
	return api.WhoopStatus200JSONResponse{Configured: h.whoop.Configured(), Connected: connected}, nil
}

// ConnectWhoop sends the browser to WHOOP to allow HomeGym access; WHOOP returns it to
// WhoopCallback.
func (h *Handler) ConnectWhoop(context.Context, api.ConnectWhoopRequestObject) (api.ConnectWhoopResponseObject, error) {
	if !h.whoop.Configured() {
		return nil, service.NotFound("WHOOP is not configured", nil)
	}
	b := make([]byte, 8)
	rand.Read(b)
	state := hex.EncodeToString(b)
	authURL, cookie := h.whoop.AuthURL(state), h.whoopStateCookie(state).String()
	return api.ConnectWhoop302Response{Headers: api.ConnectWhoop302ResponseHeaders{Location: &authURL, SetCookie: &cookie}}, nil
}

// WhoopCallback completes the connection with the code WHOOP sent and returns the browser to
// the settings page (with ?whoop=error when it failed; the cause is logged).
func (h *Handler) WhoopCallback(ctx context.Context, req api.WhoopCallbackRequestObject) (api.WhoopCallbackResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	state, code, whoopErr := value(p.State), value(p.Code), value(p.Error)
	location := settingsPage + "?whoop=error"
	switch {
	case state == "" || value(p.WhoopState) != state:
		logging.ContextError(ctx, http.StatusBadRequest, "WHOOP callback with an unknown state", nil)
	case whoopErr != "" || code == "":
		logging.ContextError(ctx, http.StatusBadRequest, "WHOOP declined the connection: "+whoopErr, nil)
	default:
		if err := h.whoop.Connect(ctx, userID, code); err != nil {
			logging.ContextError(ctx, http.StatusBadGateway, "connect WHOOP", err)
		} else {
			location = settingsPage
		}
	}
	cleared := h.whoopStateCookie("")
	cleared.MaxAge = -1
	cookie := cleared.String()
	return api.WhoopCallback302Response{Headers: api.WhoopCallback302ResponseHeaders{Location: &location, SetCookie: &cookie}}, nil
}

// whoopStateCookie is the cookie carrying the OAuth state, for 10 minutes.
func (h *Handler) whoopStateCookie(state string) *http.Cookie {
	return &http.Cookie{
		Name: whoopStateCookie, Value: state, Path: "/api/whoop", MaxAge: 600,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
	}
}

// DisconnectWhoop forgets the user's WHOOP tokens.
func (h *Handler) DisconnectWhoop(ctx context.Context, _ api.DisconnectWhoopRequestObject) (api.DisconnectWhoopResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.whoop.Disconnect(ctx, userID); err != nil {
		return nil, err
	}
	return api.DisconnectWhoop204Response{}, nil
}

// ListWhoopCandidates lists the WHOOP activities of the workout's day ([from, to), the client's
// local day) to pick the one to link.
func (h *Handler) ListWhoopCandidates(ctx context.Context, req api.ListWhoopCandidatesRequestObject) (api.ListWhoopCandidatesResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	candidates, err := h.whoop.Candidates(ctx, userID, req.Id, req.Params.From, req.Params.To)
	if err != nil {
		return nil, err
	}
	return api.ListWhoopCandidates200JSONResponse(mapSlice(candidates, func(c whoop.Candidate) api.WhoopCandidate {
		return api.WhoopCandidate{Id: c.ID, SportName: c.SportName, Start: c.Start, End: c.End, Strain: c.Strain, AvgHr: c.AvgHR}
	})), nil
}

// GetWorkoutWhoop returns the WHOOP data linked to a workout, or null.
func (h *Handler) GetWorkoutWhoop(ctx context.Context, req api.GetWorkoutWhoopRequestObject) (api.GetWorkoutWhoopResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.whoop.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return noWhoopData{}, nil
	}
	return api.GetWorkoutWhoop200JSONResponse(whoopDataToAPI(*d)), nil
}

// noWhoopData answers GetWorkoutWhoop with null: no activity is linked (the generated 200
// response can't be null).
type noWhoopData struct{}

func (noWhoopData) VisitGetWorkoutWhoopResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("null\n"))
	return err
}

// LinkWhoop links a WHOOP activity to a workout, fetching its data (again).
func (h *Handler) LinkWhoop(ctx context.Context, req api.LinkWhoopRequestObject) (api.LinkWhoopResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.whoop.Link(ctx, userID, req.Id, req.Body.WhoopWorkoutId)
	if err != nil {
		return nil, err
	}
	return api.LinkWhoop200JSONResponse(whoopDataToAPI(d)), nil
}

// UnlinkWhoop removes the WHOOP data from a workout.
func (h *Handler) UnlinkWhoop(ctx context.Context, req api.UnlinkWhoopRequestObject) (api.UnlinkWhoopResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.whoop.Unlink(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return api.UnlinkWhoop204Response{}, nil
}

func whoopDataToAPI(d whooprepos.WorkoutWhoopDatum) api.WhoopData {
	return api.WhoopData{
		WorkoutId: d.WorkoutID, WhoopWorkoutId: d.WhoopWorkoutID, SportName: d.SportName,
		StartAt: timestamp(d.StartAt), EndAt: timestamp(d.EndAt), ScoreState: d.ScoreState,
		Strain: d.Strain, AvgHr: d.AvgHr, MaxHr: d.MaxHr, Kilojoule: d.Kilojoule, ZoneMs: optionalSlice(d.ZoneMs),
		RecoveryScore: d.RecoveryScore, RestingHr: d.RestingHr, HrvMs: d.HrvMs, FetchedAt: timestamp(d.FetchedAt),
	}
}

// optionalSlice is a nullable array: nil (null) when the column is NULL.
func optionalSlice[T any](s []T) *[]T {
	if s == nil {
		return nil
	}
	return &s
}
