package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/erdira/homegym/backend/common/web"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
	"github.com/erdira/homegym/backend/modules/tracking/app"
)

// ReadModel lists what the tracker and the dashboard show, straight as the HTTP responses (no app
// layer: reads are plain data retrieval). Only the user's own data; a missing (or another user's)
// workout or 1RM is an errs.NotFound error.
type ReadModel interface {
	Workouts(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]WorkoutSummary, error)
	Workout(ctx context.Context, userID, id uuid.UUID) (Workout, error)
	Maxes(ctx context.Context, userID uuid.UUID) ([]ExerciseMax, error)
	Max(ctx context.Context, userID, id uuid.UUID) (ExerciseMax, error)
	ExerciseStats(ctx context.Context, userID uuid.UUID) ([]ExerciseStat, error)
	LastLoads(ctx context.Context, userID uuid.UUID) ([]LastLoad, error)
}

// Server implements the tracking part of the API (StrictServerInterface).
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

// GetWeek returns the week for the tracker: the active trainings, the sessions the chosen one
// (training, optional: else the first) plans for it and the workouts finished. The client sends
// its local week: the Monday as date plus the bounds from and to (instants).
func (s *Server) GetWeek(ctx context.Context, req GetWeekRequestObject) (GetWeekResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	var trainingID uuid.UUID
	if p.Training != nil {
		trainingID = *p.Training
	}
	wk, err := s.service.Week(ctx, userID, trainingID, p.Date.Time, p.From, p.To)
	if err != nil {
		return nil, err
	}
	out := GetWeek200JSONResponse{
		Trainings:  make([]ActiveTraining, len(wk.Trainings)),
		WeekNumber: wk.WeekNumber,
		Planned:    make([]PlannedSlot, len(wk.Planned)),
		Done:       make([]WorkoutSummary, len(wk.Done)),
	}
	for i, t := range wk.Trainings {
		out.Trainings[i] = activeTrainingToAPI(t)
	}
	for i, slot := range wk.Planned {
		out.Planned[i] = PlannedSlot{Id: slot.ID, SessionId: slot.SessionID, Name: slot.SessionName}
	}
	for i, w := range wk.Done {
		out.Done[i] = WorkoutSummaryToAPI(w)
	}
	if wk.Training != nil {
		t := activeTrainingToAPI(*wk.Training)
		out.Training = &t
	}
	if c := wk.Cycle; c != nil {
		out.Cycle = &WeekCycle{
			Id: c.ID, Name: c.Name, StartDate: date(c.StartDate), EndDate: date(c.EndDate), Weeks: c.Weeks, WeekNumber: c.WeekNumber,
			Goals: make([]WeekCycleGoal, len(c.Goals)),
		}
		for i, g := range c.Goals {
			out.Cycle.Goals[i] = WeekCycleGoal{Title: g.Title, Target: g.Target, Status: GoalStatus(g.Status)}
		}
	}
	return out, nil
}

// ListWorkouts returns the user's workouts finished in [from, to), newest first.
func (s *Server) ListWorkouts(ctx context.Context, req ListWorkoutsRequestObject) (ListWorkoutsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := app.CheckPeriod(req.Params.From, req.Params.To); err != nil {
		return nil, err
	}
	workouts, err := s.readModel.Workouts(ctx, userID, req.Params.From, req.Params.To)
	return ListWorkouts200JSONResponse(workouts), err
}

// StartWorkout begins a workout of a planned session (a slot of the week plan): 201, or 200 with
// the workout already in progress of that slot.
func (s *Server) StartWorkout(ctx context.Context, req StartWorkoutRequestObject) (StartWorkoutResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	id, created, err := s.service.StartWorkout(ctx, userID, req.Body.WeekSessionId)
	if err != nil {
		return nil, err
	}
	w, err := s.readModel.Workout(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if created {
		return StartWorkout201JSONResponse(w), nil
	}
	return StartWorkout200JSONResponse(w), nil
}

// GetWorkout returns one of the user's workouts with its sets and scores.
func (s *Server) GetWorkout(ctx context.Context, req GetWorkoutRequestObject) (GetWorkoutResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.readModel.Workout(ctx, userID, req.Id)
	return GetWorkout200JSONResponse(w), err
}

// FinishWorkout completes a workout with notes (or only updates the notes of a finished one),
// optionally dated to when it was done.
func (s *Server) FinishWorkout(ctx context.Context, req FinishWorkoutRequestObject) (FinishWorkoutResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.FinishWorkout(ctx, userID, req.Id, req.Body.Notes, req.Body.CompletedAt); err != nil {
		return nil, err
	}
	w, err := s.readModel.Workout(ctx, userID, req.Id)
	return FinishWorkout200JSONResponse(w), err
}

// DeleteWorkout discards a workout in progress or removes a finished one.
func (s *Server) DeleteWorkout(ctx context.Context, req DeleteWorkoutRequestObject) (DeleteWorkoutResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteWorkout(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return DeleteWorkout204Response{}, nil
}

// LogSet stores a set as it was done, or corrects one logged before.
func (s *Server) LogSet(ctx context.Context, req LogSetRequestObject) (LogSetResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	set, err := s.service.LogSet(ctx, userID, req.Id, app.SetInput{
		ExerciseID: b.ExerciseId, BlockPosition: b.BlockPosition, ExercisePosition: b.ExercisePosition, SetNumber: b.SetNumber,
		RepsValue: b.RepsValue, RepsUnit: b.RepsUnit, PerSide: b.PerSide, LoadValue: b.LoadValue, LoadUnit: b.LoadUnit,
		Bodyweight: b.Bodyweight,
	})
	if err != nil {
		return nil, err
	}
	return LogSet200JSONResponse{
		Id: set.ID, WorkoutId: set.WorkoutID, ExerciseId: set.ExerciseID,
		BlockPosition: set.BlockPosition, ExercisePosition: set.ExercisePosition, SetNumber: set.SetNumber,
		RepsValue: set.RepsValue, RepsUnit: set.RepsUnit, PerSide: set.PerSide, LoadValue: set.LoadValue, LoadUnit: set.LoadUnit,
		Bodyweight: set.Bodyweight, CompletedAt: set.CompletedAt,
	}, nil
}

// DeleteSet removes a set from a workout.
func (s *Server) DeleteSet(ctx context.Context, req DeleteSetRequestObject) (DeleteSetResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteSet(ctx, userID, req.Id, req.SetID); err != nil {
		return nil, err
	}
	return DeleteSet204Response{}, nil
}

// LogResult stores the score of a WOD block as it was done, or corrects one logged before.
func (s *Server) LogResult(ctx context.Context, req LogResultRequestObject) (LogResultResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	r, err := s.service.LogResult(ctx, userID, req.Id, app.ResultInput{
		BlockPosition: b.BlockPosition, ScoreType: string(b.ScoreType), TimeSeconds: b.TimeSeconds,
		Rounds: b.Rounds, Reps: b.Reps, Calories: b.Calories, Rx: b.Rx, Notes: b.Notes,
	})
	if err != nil {
		return nil, err
	}
	return LogResult200JSONResponse{
		Id: r.ID, WorkoutId: r.WorkoutID, BlockPosition: r.BlockPosition, ScoreType: ScoreType(r.ScoreType),
		TimeSeconds: r.TimeSeconds, Rounds: r.Rounds, Reps: r.Reps, Calories: r.Calories, Rx: r.Rx, Notes: r.Notes,
		CompletedAt: r.CompletedAt,
	}, nil
}

// DeleteResult removes the score of a WOD block from a workout.
func (s *Server) DeleteResult(ctx context.Context, req DeleteResultRequestObject) (DeleteResultResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteResult(ctx, userID, req.Id, req.ResultID); err != nil {
		return nil, err
	}
	return DeleteResult204Response{}, nil
}

// AddExerciseMax adds a 1RM to the history of one of the user's exercises; the newest is the
// current max.
func (s *Server) AddExerciseMax(ctx context.Context, req AddExerciseMaxRequestObject) (AddExerciseMaxResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	id, err := s.service.AddMax(ctx, userID, req.Id, app.MaxInput{
		LoadValue: b.LoadValue, LoadUnit: b.LoadUnit, AchievedOn: b.AchievedOn, WorkoutSetID: b.WorkoutSetId, Notes: b.Notes,
	})
	if err != nil {
		return nil, err
	}
	m, err := s.readModel.Max(ctx, userID, id)
	return AddExerciseMax201JSONResponse(m), err
}

// ListExerciseMaxes returns the user's 1RM history of all their exercises, newest first.
func (s *Server) ListExerciseMaxes(ctx context.Context, _ ListExerciseMaxesRequestObject) (ListExerciseMaxesResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	maxes, err := s.readModel.Maxes(ctx, userID)
	return ListExerciseMaxes200JSONResponse(maxes), err
}

// DeleteExerciseMax removes an entry from the user's 1RM history.
func (s *Server) DeleteExerciseMax(ctx context.Context, req DeleteExerciseMaxRequestObject) (DeleteExerciseMaxResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteMax(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return DeleteExerciseMax204Response{}, nil
}

// ExerciseStats lists the exercises the user has done in workouts, most often first, with their
// rating of each.
func (s *Server) ExerciseStats(ctx context.Context, _ ExerciseStatsRequestObject) (ExerciseStatsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.readModel.ExerciseStats(ctx, userID)
	return ExerciseStats200JSONResponse(stats), err
}

// ListLastLoads returns the load the user last logged of each exercise, the tracker's suggestion
// where the plan has none.
func (s *Server) ListLastLoads(ctx context.Context, _ ListLastLoadsRequestObject) (ListLastLoadsResponseObject, error) {
	userID, err := web.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	loads, err := s.readModel.LastLoads(ctx, userID)
	return ListLastLoads200JSONResponse(loads), err
}

// WorkoutSummaryToAPI maps a finished workout as listed.
func WorkoutSummaryToAPI(w app.WorkoutSummary) WorkoutSummary {
	return WorkoutSummary{
		Id: w.ID, TrainingId: w.TrainingID, WeekSessionId: w.WeekSessionID, SessionId: w.SessionID, SessionName: w.SessionName,
		StartedAt: w.StartedAt, CompletedAt: w.CompletedAt, Notes: w.Notes, SetCount: w.SetCount,
	}
}

func activeTrainingToAPI(t planning.ActiveTraining) ActiveTraining {
	return ActiveTraining{
		Id: t.ID, Name: t.Name, StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks,
		CycleId: t.CycleID, CycleName: t.CycleName,
	}
}

// date is a date as an API date.
func date(t time.Time) openapi_types.Date { return openapi_types.Date{Time: t} }
