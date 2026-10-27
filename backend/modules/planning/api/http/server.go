package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/common/errs"
	"github.com/erdira/homegym/backend/common/web"
	"github.com/erdira/homegym/backend/modules/planning/app"
)

// ReadModel lists what the dashboard and forms show, straight as the HTTP responses (no app
// layer: reads are plain data retrieval). Only the user's own data; a missing (or another user's)
// training, template or cycle is an errs.NotFound error.
type ReadModel interface {
	Trainings(ctx context.Context, userID uuid.UUID) ([]Training, error)
	Training(ctx context.Context, userID, id uuid.UUID) (TrainingDetail, error)
	LibrarySessions(ctx context.Context, userID uuid.UUID) ([]LibrarySession, error)
	LibrarySession(ctx context.Context, userID, id uuid.UUID) (LibrarySessionDetail, error)
	Cycles(ctx context.Context, userID uuid.UUID) ([]CycleSummary, error)
	Cycle(ctx context.Context, userID, id uuid.UUID) (Cycle, error)
}

// Server implements the planning part of the API (StrictServerInterface).
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

// maxImportBody is the body limit of a cycle import: a 10-week cycle with ids is a few hundred KB.
const maxImportBody = 8 << 20

// Register mounts the routes on r (the /api router), behind requireUser.
func Register(r chi.Router, s *Server, requireUser func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(requireUser, limitBody)
		strict := NewStrictHandlerWithOptions(s, nil, StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  web.BodyError,
			ResponseErrorHandlerFunc: web.ServiceError,
		})
		HandlerWithOptions(strict, ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: paramError})
	})
}

// limitBody caps bodies at web.MaxBody, a cycle import's at maxImportBody.
func limitBody(next http.Handler) http.Handler {
	small, large := web.LimitBody(web.MaxBody)(next), web.LimitBody(maxImportBody)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/cycles/import" {
			large.ServeHTTP(w, r)
			return
		}
		small.ServeHTTP(w, r)
	})
}

// paramError answers a path or query parameter not of the spec's type (an id that isn't a UUID).
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

// ListTrainings returns the user's trainings with their size, for the overview.
func (s *Server) ListTrainings(ctx context.Context, _ ListTrainingsRequestObject) (ListTrainingsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	trainings, err := s.readModel.Trainings(ctx, userID)
	return ListTrainings200JSONResponse(trainings), err
}

// GetTraining returns one of the user's trainings with its sessions and schedule.
func (s *Server) GetTraining(ctx context.Context, req GetTrainingRequestObject) (GetTrainingResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.readModel.Training(ctx, userID, req.Id)
	return GetTraining200JSONResponse(t), err
}

// CreateTraining saves a new training with its whole tree.
func (s *Server) CreateTraining(ctx context.Context, req CreateTrainingRequestObject) (CreateTrainingResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.service.CreateTraining(ctx, userID, trainingInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return CreateTraining201JSONResponse(trainingToAPI(t)), nil
}

// UpdateTraining replaces a training's fields and tree.
func (s *Server) UpdateTraining(ctx context.Context, req UpdateTrainingRequestObject) (UpdateTrainingResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.service.UpdateTraining(ctx, userID, req.Id, trainingInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return UpdateTraining200JSONResponse(trainingToAPI(t)), nil
}

// DuplicateTraining copies a training as the next phase, inactive.
func (s *Server) DuplicateTraining(ctx context.Context, req DuplicateTrainingRequestObject) (DuplicateTrainingResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.service.DuplicateTraining(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return DuplicateTraining201JSONResponse(trainingToAPI(t)), nil
}

// DeleteTraining soft-deletes a training.
func (s *Server) DeleteTraining(ctx context.Context, req DeleteTrainingRequestObject) (DeleteTrainingResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteTraining(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return DeleteTraining204Response{}, nil
}

// ListSessions returns the user's session templates with their size, for the library.
func (s *Server) ListSessions(ctx context.Context, _ ListSessionsRequestObject) (ListSessionsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := s.readModel.LibrarySessions(ctx, userID)
	return ListSessions200JSONResponse(sessions), err
}

// GetSession returns one of the user's session templates with its blocks and exercises.
func (s *Server) GetSession(ctx context.Context, req GetSessionRequestObject) (GetSessionResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.readModel.LibrarySession(ctx, userID, req.Id)
	return GetSession200JSONResponse(session), err
}

// CreateSession saves a new session template with its blocks.
func (s *Server) CreateSession(ctx context.Context, req CreateSessionRequestObject) (CreateSessionResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.service.CreateSession(ctx, userID, sessionInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	session, err := s.readModel.LibrarySession(ctx, userID, id)
	return CreateSession201JSONResponse(session), err
}

// UpdateSession replaces a session template's fields and blocks; trainings keep their copies.
func (s *Server) UpdateSession(ctx context.Context, req UpdateSessionRequestObject) (UpdateSessionResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.UpdateSession(ctx, userID, req.Id, sessionInputFromAPI(*req.Body)); err != nil {
		return nil, err
	}
	session, err := s.readModel.LibrarySession(ctx, userID, req.Id)
	return UpdateSession200JSONResponse(session), err
}

// DeleteSession soft-deletes a session template; trainings keep their copies.
func (s *Server) DeleteSession(ctx context.Context, req DeleteSessionRequestObject) (DeleteSessionResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteSession(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return DeleteSession204Response{}, nil
}

// ListCycles returns the user's cycles with their goal and training counts.
func (s *Server) ListCycles(ctx context.Context, _ ListCyclesRequestObject) (ListCyclesResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	cycles, err := s.readModel.Cycles(ctx, userID)
	return ListCycles200JSONResponse(cycles), err
}

// GetCycle returns one of the user's cycles with its goals and trainings.
func (s *Server) GetCycle(ctx context.Context, req GetCycleRequestObject) (GetCycleResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.readModel.Cycle(ctx, userID, req.Id)
	return GetCycle200JSONResponse(c), err
}

// CreateCycle saves a new cycle with its goals.
func (s *Server) CreateCycle(ctx context.Context, req CreateCycleRequestObject) (CreateCycleResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.service.CreateCycle(ctx, userID, cycleInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	c, err := s.readModel.Cycle(ctx, userID, id)
	return CreateCycle201JSONResponse(c), err
}

// UpdateCycle replaces a cycle's fields and goals.
func (s *Server) UpdateCycle(ctx context.Context, req UpdateCycleRequestObject) (UpdateCycleResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.UpdateCycle(ctx, userID, req.Id, cycleInputFromAPI(*req.Body)); err != nil {
		return nil, err
	}
	c, err := s.readModel.Cycle(ctx, userID, req.Id)
	return UpdateCycle200JSONResponse(c), err
}

// DeleteCycle soft-deletes a cycle. ?trainings=delete deletes its trainings too; by default
// (keep) they stay and just leave the cycle.
func (s *Server) DeleteCycle(ctx context.Context, req DeleteCycleRequestObject) (DeleteCycleResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	var withTrainings bool
	if t := req.Params.Trainings; t != nil {
		switch *t {
		case Keep:
		case Delete:
			withTrainings = true
		default:
			return nil, errs.Invalid("trainings must be keep or delete")
		}
	}
	if err := s.service.DeleteCycle(ctx, userID, req.Id, withTrainings); err != nil {
		return nil, err
	}
	return DeleteCycle204Response{}, nil
}

// ImportCycle saves a whole cycle with its trainings at once, or nothing if any part is invalid.
func (s *Server) ImportCycle(ctx context.Context, req ImportCycleRequestObject) (ImportCycleResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	trainings := make([]app.Input, len(req.Body.Trainings))
	for i, t := range req.Body.Trainings {
		trainings[i] = trainingInputFromAPI(t)
	}
	c, err := s.service.ImportCycle(ctx, userID, cycleInputFromAPI(req.Body.Cycle), trainings)
	if err != nil {
		return nil, err
	}
	out := ImportCycle201JSONResponse{
		Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
		Weeks: c.Weeks, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Trainings: make([]TrainingSaved, len(c.Trainings)),
	}
	for i, t := range c.Trainings {
		out.Trainings[i] = trainingToAPI(t)
	}
	return out, nil
}

func trainingToAPI(t app.TrainingHeader) TrainingSaved {
	return TrainingSaved{
		Id: t.ID, CycleId: t.CycleID, Name: t.Name, Description: t.Description, Active: t.Active,
		StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func trainingInputFromAPI(in TrainingInput) app.Input {
	out := app.Input{
		CycleID: in.CycleId, Name: in.Name, Description: in.Description, Active: in.Active,
		StartDate: in.StartDate, Weeks: in.Weeks,
		Sessions: make([]app.SessionInput, len(in.Sessions)), Schedule: make([][]app.SlotInput, len(in.Schedule)),
	}
	for i, s := range in.Sessions {
		out.Sessions[i] = sessionInputFromAPI(s)
	}
	for w, week := range in.Schedule {
		out.Schedule[w] = make([]app.SlotInput, len(week))
		for i, slot := range week {
			out.Schedule[w][i] = app.SlotInput{ID: slot.Id, Session: slot.Session}
		}
	}
	return out
}

func sessionInputFromAPI(s TrainingSessionInput) app.SessionInput {
	out := app.SessionInput{ID: s.Id, Name: s.Name, Notes: s.Notes, Blocks: make([]app.BlockInput, len(s.Blocks))}
	for i, b := range s.Blocks {
		block := app.BlockInput{
			Kind: b.Kind, Name: b.Name, Sets: b.Sets,
			Amount: app.Amount{
				RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
				LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, LoadBodyweight: b.LoadBodyweight,
			},
			Score:     app.Score{ScoreType: (*string)(b.ScoreType), TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds},
			Notes:     b.Notes,
			Exercises: make([]app.ExerciseInput, len(b.Exercises)),
		}
		for j, e := range b.Exercises {
			block.Exercises[j] = app.ExerciseInput{
				ExerciseID: e.ExerciseId, Sets: e.Sets,
				Amount: app.Amount{
					RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
					LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, LoadBodyweight: e.LoadBodyweight,
				},
				Tempo: e.Tempo, Notes: e.Notes,
			}
		}
		out.Blocks[i] = block
	}
	return out
}

func cycleInputFromAPI(in CycleInput) app.CycleInput {
	out := app.CycleInput{
		Name: in.Name, Description: in.Description, StartDate: in.StartDate, Weeks: in.Weeks,
		Goals: make([]app.GoalInput, len(in.Goals)),
	}
	for i, g := range in.Goals {
		out.Goals[i] = app.GoalInput{ID: g.Id, Title: g.Title, Target: g.Target, Status: string(g.Status), Notes: g.Notes}
	}
	return out
}

// date is a date as an API date.
func date(t time.Time) openapi_types.Date { return openapi_types.Date{Time: t} }
