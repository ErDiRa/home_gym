package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/nutrition/app"
)

// ReadModel lists the users' bodies straight as the HTTP responses (no app layer: reads are plain
// data retrieval). Only the user's own data.
type ReadModel interface {
	// Profile returns the user's profile, or nil.
	Profile(ctx context.Context, userID uuid.UUID) (*Profile, error)
	Weights(ctx context.Context, userID uuid.UUID) ([]BodyWeight, error)
}

// Server implements the nutrition part of the API (StrictServerInterface).
type Server struct {
	service   *app.Service
	readModel ReadModel
}

var _ StrictServerInterface = (*Server)(nil)

func NewServer(service *app.Service, readModel ReadModel) *Server {
	if service == nil {
		panic("service cannot be nil")
	}
	if readModel == nil {
		panic("readModel cannot be nil")
	}
	return &Server{service: service, readModel: readModel}
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

// GetNutritionProfile returns the user's body, or null before it was entered.
func (s *Server) GetNutritionProfile(ctx context.Context, _ GetNutritionProfileRequestObject) (GetNutritionProfileResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.readModel.Profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return noProfile{}, nil
	}
	return GetNutritionProfile200JSONResponse(*p), nil
}

// noProfile answers GetNutritionProfile with null: none was entered (the generated 200 response
// can't be null).
type noProfile struct{}

func (noProfile) VisitGetNutritionProfileResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("null\n"))
	return err
}

// SaveNutritionProfile sets the user's body.
func (s *Server) SaveNutritionProfile(ctx context.Context, req SaveNutritionProfileRequestObject) (SaveNutritionProfileResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	in := app.ProfileInput{HeightCm: b.HeightCm, BirthDate: b.BirthDate, Sex: app.Sex(b.Sex)}
	if b.ActivityFactor != nil {
		in.ActivityFactor = *b.ActivityFactor
	}
	if err := s.service.SaveProfile(ctx, userID, in); err != nil {
		return nil, err
	}
	p, err := s.readModel.Profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return SaveNutritionProfile200JSONResponse(*p), nil
}

// ListBodyWeights lists the user's weights, newest first.
func (s *Server) ListBodyWeights(ctx context.Context, _ ListBodyWeightsRequestObject) (ListBodyWeightsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	weights, err := s.readModel.Weights(ctx, userID)
	if err != nil {
		return nil, err
	}
	return ListBodyWeights200JSONResponse(weights), nil
}

// SaveBodyWeight logs the weight (and body fat) of a day.
func (s *Server) SaveBodyWeight(ctx context.Context, req SaveBodyWeightRequestObject) (SaveBodyWeightResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	w, err := s.service.SaveWeight(ctx, userID, app.WeightInput{
		MeasuredOn: b.MeasuredOn, WeightKg: b.WeightKg, BodyFatPercent: b.BodyFatPercent, Notes: value(b.Notes),
	})
	if err != nil {
		return nil, err
	}
	return SaveBodyWeight200JSONResponse{
		Id: w.ID, MeasuredOn: openapi_types.Date{Time: w.MeasuredOn}, WeightKg: w.WeightKg, BodyFatPercent: w.BodyFatPercent, Notes: w.Notes,
	}, nil
}

// DeleteBodyWeight removes an entry of the weight history.
func (s *Server) DeleteBodyWeight(ctx context.Context, req DeleteBodyWeightRequestObject) (DeleteBodyWeightResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteWeight(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return DeleteBodyWeight204Response{}, nil
}

// ListNutritionDays returns the user's days with their macros, targets and workouts.
func (s *Server) ListNutritionDays(ctx context.Context, req ListNutritionDaysRequestObject) (ListNutritionDaysResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	days, err := s.service.Days(ctx, userID, p.From, p.To, p.Tz)
	if err != nil {
		return nil, err
	}
	out := make(ListNutritionDays200JSONResponse, len(days))
	for i, d := range days {
		out[i] = NutritionDay{Date: openapi_types.Date{Time: d.Date}, Workouts: make([]DayWorkout, len(d.Workouts))}
		if d.Macros != nil {
			m := macrosToAPI(*d.Macros)
			out[i].Macros = &m
		}
		if t := d.Target; t != nil {
			out[i].Target = &MacroTarget{
				Kcal: t.Kcal, ProteinG: t.ProteinG, FatG: t.FatG, CarbsG: t.CarbsG,
				Source: EnergySource(t.Source), WeightKg: t.WeightKg,
			}
		}
		for j, w := range d.Workouts {
			out[i].Workouts[j] = DayWorkout{Id: w.ID, SessionName: w.SessionName, StartedAt: w.StartedAt, CompletedAt: w.CompletedAt}
		}
	}
	return out, nil
}

// SaveDayMacros logs the macros eaten on a day.
func (s *Server) SaveDayMacros(ctx context.Context, req SaveDayMacrosRequestObject) (SaveDayMacrosResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	m, err := s.service.SaveMacros(ctx, userID, req.Date, app.MacrosInput{
		ProteinG: b.ProteinG, FatG: b.FatG, CarbsG: b.CarbsG, Notes: value(b.Notes),
	})
	if err != nil {
		return nil, err
	}
	return SaveDayMacros200JSONResponse(macrosToAPI(m)), nil
}

// DeleteDayMacros removes the macros logged for a day.
func (s *Server) DeleteDayMacros(ctx context.Context, req DeleteDayMacrosRequestObject) (DeleteDayMacrosResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteMacros(ctx, userID, req.Date); err != nil {
		return nil, err
	}
	return DeleteDayMacros204Response{}, nil
}

func macrosToAPI(m app.Macros) DayMacros {
	return DayMacros{
		Date: openapi_types.Date{Time: m.Day}, ProteinG: m.ProteinG, FatG: m.FatG, CarbsG: m.CarbsG,
		Kcal: m.Kcal, Notes: m.Notes, UpdatedAt: m.Updated,
	}
}

// value returns *p, or "" when p is nil (an optional field the request left out).
func value(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
