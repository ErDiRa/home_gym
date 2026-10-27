package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/catalog/app"
)

// ReadModel lists what the dashboard and forms show, straight as the HTTP responses (no app
// layer: reads are plain data retrieval). Only the user's own exercises and ratings.
type ReadModel interface {
	Exercises(ctx context.Context, userID uuid.UUID) ([]Exercise, error)
	// Exercise returns an errs.NotFound error for a missing exercise (or another user's).
	Exercise(ctx context.Context, userID, id uuid.UUID) (ExerciseDetail, error)
	Ratings(ctx context.Context, userID uuid.UUID) ([]ExerciseRating, error)
	Joints(ctx context.Context) ([]Option, error)
	MuscleGroups(ctx context.Context) ([]Option, error)
	Focuses(ctx context.Context) ([]Focus, error)
	BlockKinds(ctx context.Context) ([]BlockKind, error)
	RepUnits(ctx context.Context) ([]Unit, error)
	LoadUnits(ctx context.Context) ([]LoadUnit, error)
}

// Server implements the catalog part of the API (StrictServerInterface).
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

// paramError answers a path parameter not of the spec's type (an id that isn't a UUID).
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

// ListExercises returns the user's exercises with the names of their links.
func (s *Server) ListExercises(ctx context.Context, _ ListExercisesRequestObject) (ListExercisesResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	exercises, err := s.readModel.Exercises(ctx, userID)
	return ListExercises200JSONResponse(exercises), err
}

// GetExercise returns one of the user's exercises with the ids of its links, for the form.
func (s *Server) GetExercise(ctx context.Context, req GetExerciseRequestObject) (GetExerciseResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.readModel.Exercise(ctx, userID, req.Id)
	return GetExercise200JSONResponse(e), err
}

func (s *Server) CreateExercise(ctx context.Context, req CreateExerciseRequestObject) (CreateExerciseResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.service.CreateExercise(ctx, userID, exerciseInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return CreateExercise201JSONResponse(exerciseToAPI(e)), nil
}

func (s *Server) UpdateExercise(ctx context.Context, req UpdateExerciseRequestObject) (UpdateExerciseResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.service.UpdateExercise(ctx, userID, req.Id, exerciseInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return UpdateExercise200JSONResponse(exerciseToAPI(e)), nil
}

// RateExercise stores the user's rating (1-5) and notes for one of their exercises, replacing any
// earlier one.
func (s *Server) RateExercise(ctx context.Context, req RateExerciseRequestObject) (RateExerciseResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.service.RateExercise(ctx, userID, req.Id, app.RatingInput{Rating: req.Body.Rating, Notes: req.Body.Notes})
	if err != nil {
		return nil, err
	}
	return RateExercise200JSONResponse{
		Id: r.ID, UserId: r.UserID, ExerciseId: r.ExerciseID, Rating: r.Rating, Notes: r.Notes, UpdatedAt: r.UpdatedAt,
	}, nil
}

// ListExerciseRatings returns how the user rated their exercises, for the exercise picker.
func (s *Server) ListExerciseRatings(ctx context.Context, _ ListExerciseRatingsRequestObject) (ListExerciseRatingsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	ratings, err := s.readModel.Ratings(ctx, userID)
	return ListExerciseRatings200JSONResponse(ratings), err
}

func (s *Server) ListJoints(ctx context.Context, _ ListJointsRequestObject) (ListJointsResponseObject, error) {
	joints, err := s.readModel.Joints(ctx)
	return ListJoints200JSONResponse(joints), err
}

func (s *Server) ListMuscleGroups(ctx context.Context, _ ListMuscleGroupsRequestObject) (ListMuscleGroupsResponseObject, error) {
	groups, err := s.readModel.MuscleGroups(ctx)
	return ListMuscleGroups200JSONResponse(groups), err
}

func (s *Server) ListFocuses(ctx context.Context, _ ListFocusesRequestObject) (ListFocusesResponseObject, error) {
	focuses, err := s.readModel.Focuses(ctx)
	return ListFocuses200JSONResponse(focuses), err
}

func (s *Server) CreateFocus(ctx context.Context, req CreateFocusRequestObject) (CreateFocusResponseObject, error) {
	f, err := s.service.CreateFocus(ctx, app.FocusInput{Name: req.Body.Name, Alias: req.Body.Alias})
	if err != nil {
		return nil, err
	}
	return CreateFocus201JSONResponse{Id: f.ID, Name: f.Name, Alias: f.Alias}, nil
}

// Block kinds and the units of planned reps and load, which trainings are validated against.

func (s *Server) ListBlockKinds(ctx context.Context, _ ListBlockKindsRequestObject) (ListBlockKindsResponseObject, error) {
	kinds, err := s.readModel.BlockKinds(ctx)
	return ListBlockKinds200JSONResponse(kinds), err
}

func (s *Server) ListRepUnits(ctx context.Context, _ ListRepUnitsRequestObject) (ListRepUnitsResponseObject, error) {
	units, err := s.readModel.RepUnits(ctx)
	return ListRepUnits200JSONResponse(units), err
}

func (s *Server) ListLoadUnits(ctx context.Context, _ ListLoadUnitsRequestObject) (ListLoadUnitsResponseObject, error) {
	units, err := s.readModel.LoadUnits(ctx)
	return ListLoadUnits200JSONResponse(units), err
}

func (s *Server) CreateRepUnit(ctx context.Context, req CreateRepUnitRequestObject) (CreateRepUnitResponseObject, error) {
	u, err := s.service.CreateRepUnit(ctx, unitInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return CreateRepUnit201JSONResponse{Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue}, nil
}

func (s *Server) CreateLoadUnit(ctx context.Context, req CreateLoadUnitRequestObject) (CreateLoadUnitResponseObject, error) {
	u, err := s.service.CreateLoadUnit(ctx, unitInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return CreateLoadUnit201JSONResponse{
		Id: u.ID, Name: u.Name, MinValue: u.MinValue, MaxValue: u.MaxValue, KgPerUnit: u.KgPerUnit, PercentOfMax: u.PercentOfMax,
	}, nil
}

func exerciseInputFromAPI(in ExerciseInput) app.ExerciseInput {
	return app.ExerciseInput{
		Name: in.Name, Type: string(in.Type), Description: in.Description, VideoURLs: in.VideoUrls,
		JointIDs: in.JointIds, MuscleGroupIDs: in.MuscleGroupIds, FocusIDs: in.FocusIds,
	}
}

func exerciseToAPI(e app.Exercise) ExerciseSaved {
	return ExerciseSaved{
		Id: e.ID, Name: e.Name, Description: e.Description, Type: ExerciseType(e.Type), VideoUrls: e.VideoURLs,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

func unitInputFromAPI(in UnitInput) app.UnitInput {
	return app.UnitInput{Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue}
}
