package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/erdira/homegym/backend/common/errs"
	"github.com/erdira/homegym/backend/common/log"
	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/whoop/app"
)

// whoopStateCookie carries the OAuth state from ConnectWhoop to WhoopCallback, so the callback
// only accepts a code for the connection this browser started.
const whoopStateCookie = "whoop_state"

// settingsPage is where the browser returns to after connecting WHOOP.
const settingsPage = "/track/settings"

// Server implements the whoop part of the API (StrictServerInterface).
type Server struct {
	service *app.Service
	// secureCookies marks the state cookie Secure (HTTPS only); off for local HTTP dev.
	secureCookies bool
}

var _ StrictServerInterface = (*Server)(nil)

func NewServer(service *app.Service, secureCookies bool) *Server {
	if service == nil {
		panic("service cannot be nil")
	}
	return &Server{service: service, secureCookies: secureCookies}
}

// Register mounts the routes on r (the /api router), behind requireUser.
func Register(r chi.Router, s *Server, requireUser func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireUser, web.LimitBody(web.MaxBody))
		strict := NewStrictHandlerWithOptions(s, nil, StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  web.BodyError,
			ResponseErrorHandlerFunc: web.ServiceError,
		})
		HandlerWithOptions(strict, ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: paramError})
	})
}

// paramError answers a path or query parameter that is missing or not of the spec's type.
func paramError(w http.ResponseWriter, r *http.Request, err error) {
	var required *RequiredParamError
	var invalid *InvalidParamFormatError
	switch {
	case errors.As(err, &required):
		web.MissingParam(w, r, required.ParamName, err)
	case errors.As(err, &invalid):
		web.InvalidParam(w, r, invalid.ParamName, err)
	default:
		web.ClientError(w, r, http.StatusBadRequest, "invalid request", err)
	}
}

// WhoopStatus tells whether WHOOP can be connected (the app is configured) and whether it is.
func (s *Server) WhoopStatus(ctx context.Context, _ WhoopStatusRequestObject) (WhoopStatusResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	connected, err := s.service.Connected(ctx, userID)
	if err != nil {
		return nil, err
	}
	return WhoopStatus200JSONResponse{Configured: s.service.Configured(), Connected: connected}, nil
}

// ConnectWhoop sends the browser to WHOOP to allow HomeGym access; WHOOP returns it to
// WhoopCallback.
func (s *Server) ConnectWhoop(context.Context, ConnectWhoopRequestObject) (ConnectWhoopResponseObject, error) {
	if !s.service.Configured() {
		return nil, errs.NotFound("WHOOP is not configured", nil)
	}
	b := make([]byte, 8)
	rand.Read(b)
	state := hex.EncodeToString(b)
	authURL, cookie := s.service.AuthURL(state), s.stateCookie(state).String()
	return ConnectWhoop302Response{Headers: ConnectWhoop302ResponseHeaders{Location: &authURL, SetCookie: &cookie}}, nil
}

// WhoopCallback completes the connection with the code WHOOP sent and returns the browser to the
// settings page (with ?whoop=error when it failed; the cause is logged).
func (s *Server) WhoopCallback(ctx context.Context, req WhoopCallbackRequestObject) (WhoopCallbackResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	state, code, whoopErr := value(p.State), value(p.Code), value(p.Error)
	location := settingsPage + "?whoop=error"
	switch {
	case state == "" || value(p.WhoopState) != state:
		log.ContextError(ctx, http.StatusBadRequest, "WHOOP callback with an unknown state", nil)
	case whoopErr != "" || code == "":
		log.ContextError(ctx, http.StatusBadRequest, "WHOOP declined the connection: "+whoopErr, nil)
	default:
		if err := s.service.Connect(ctx, userID, code); err != nil {
			log.ContextError(ctx, http.StatusBadGateway, "connect WHOOP", err)
		} else {
			location = settingsPage
		}
	}
	cleared := s.stateCookie("")
	cleared.MaxAge = -1
	cookie := cleared.String()
	return WhoopCallback302Response{Headers: WhoopCallback302ResponseHeaders{Location: &location, SetCookie: &cookie}}, nil
}

// stateCookie is the cookie carrying the OAuth state, for 10 minutes.
func (s *Server) stateCookie(state string) *http.Cookie {
	return &http.Cookie{
		Name: whoopStateCookie, Value: state, Path: "/api/whoop", MaxAge: 600,
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	}
}

// DisconnectWhoop forgets the user's WHOOP tokens.
func (s *Server) DisconnectWhoop(ctx context.Context, _ DisconnectWhoopRequestObject) (DisconnectWhoopResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.Disconnect(ctx, userID); err != nil {
		return nil, err
	}
	return DisconnectWhoop204Response{}, nil
}

// ListWhoopCandidates lists the WHOOP activities of the workout's day ([from, to), the client's
// local day) to pick the one to link.
func (s *Server) ListWhoopCandidates(ctx context.Context, req ListWhoopCandidatesRequestObject) (ListWhoopCandidatesResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	activities, err := s.service.Candidates(ctx, userID, req.Id, req.Params.From, req.Params.To)
	if err != nil {
		return nil, err
	}
	out := make(ListWhoopCandidates200JSONResponse, len(activities))
	for i, a := range activities {
		out[i] = WhoopCandidate{Id: a.ID, SportName: a.SportName, Start: a.Start, End: a.End, Strain: a.Strain, AvgHr: a.AvgHR}
	}
	return out, nil
}

// GetWorkoutWhoop returns the WHOOP data linked to a workout, or null.
func (s *Server) GetWorkoutWhoop(ctx context.Context, req GetWorkoutWhoopRequestObject) (GetWorkoutWhoopResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.service.Data(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return noWhoopData{}, nil
	}
	return GetWorkoutWhoop200JSONResponse(dataToAPI(*d)), nil
}

// noWhoopData answers GetWorkoutWhoop with null: no activity is linked (the generated 200 response
// can't be null).
type noWhoopData struct{}

func (noWhoopData) VisitGetWorkoutWhoopResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("null\n"))
	return err
}

// LinkWhoop links a WHOOP activity to a workout, fetching its data (again).
func (s *Server) LinkWhoop(ctx context.Context, req LinkWhoopRequestObject) (LinkWhoopResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.service.Link(ctx, userID, req.Id, req.Body.WhoopWorkoutId)
	if err != nil {
		return nil, err
	}
	return LinkWhoop200JSONResponse(dataToAPI(d)), nil
}

// UnlinkWhoop removes the WHOOP data from a workout.
func (s *Server) UnlinkWhoop(ctx context.Context, req UnlinkWhoopRequestObject) (UnlinkWhoopResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.Unlink(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return UnlinkWhoop204Response{}, nil
}

func dataToAPI(d app.Data) WhoopData {
	out := WhoopData{
		WorkoutId: d.WorkoutID, WhoopWorkoutId: d.ID, SportName: d.SportName, StartAt: d.Start, EndAt: d.End,
		ScoreState: d.ScoreState, Strain: d.Strain, AvgHr: d.AvgHR, MaxHr: d.MaxHR, Kilojoule: d.Kilojoule,
		FetchedAt: d.FetchedAt,
	}
	if d.ZoneMs != nil {
		out.ZoneMs = &d.ZoneMs
	}
	if r := d.Recovery; r != nil {
		out.RecoveryScore, out.RestingHr, out.HrvMs = &r.Score, &r.RestingHR, &r.HrvMs
	}
	return out
}

// value returns *p, or "" when p is nil (an optional parameter the request left out).
func value(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
