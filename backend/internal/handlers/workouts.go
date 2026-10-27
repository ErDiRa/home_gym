package handlers

import (
	"context"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service/workout"
	workoutrepos "github.com/erdira/homegym/backend/internal/service/workout/repos"
)

// GetWeek returns the week for the tracker: the active trainings, the sessions the chosen one
// (training, optional: else the first) plans for it and the workouts finished. The client sends
// its local week: the Monday as date plus the bounds from and to (instants).
func (h *Handler) GetWeek(ctx context.Context, req api.GetWeekRequestObject) (api.GetWeekResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	p := req.Params
	wk, err := h.workouts.Week(ctx, userID, value(p.Training), p.Date.Time, p.From, p.To)
	if err != nil {
		return nil, err
	}
	out := api.GetWeek200JSONResponse{
		Trainings:  mapSlice(wk.Trainings, activeTrainingToAPI),
		WeekNumber: wk.WeekNumber,
		Planned: mapSlice(wk.Planned, func(s workoutrepos.ListWeekPlanRow) api.PlannedSlot {
			return api.PlannedSlot{Id: s.ID, SessionId: s.SessionID, Name: s.Name}
		}),
		Done: mapSlice(wk.Done, workoutSummaryToAPI),
	}
	if wk.Training != nil {
		t := activeTrainingToAPI(*wk.Training)
		out.Training = &t
	}
	if c := wk.Cycle; c != nil {
		out.Cycle = &api.WeekCycle{
			Id: c.ID, Name: c.Name, StartDate: date(c.StartDate), EndDate: date(c.EndDate), Weeks: c.Weeks, WeekNumber: c.WeekNumber,
			Goals: mapSlice(c.Goals, func(g workoutrepos.ListWeekCycleGoalsRow) api.WeekCycleGoal {
				return api.WeekCycleGoal{Title: g.Title, Target: g.Target, Status: api.GoalStatus(g.Status)}
			}),
		}
	}
	return out, nil
}

// ListWorkouts returns the user's workouts finished in [from, to), newest first.
func (h *Handler) ListWorkouts(ctx context.Context, req api.ListWorkoutsRequestObject) (api.ListWorkoutsResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.workouts.List(ctx, userID, req.Params.From, req.Params.To)
	if err != nil {
		return nil, err
	}
	return api.ListWorkouts200JSONResponse(mapSlice(rows, workoutSummaryToAPI)), nil
}

// StartWorkout begins a workout of a planned session (a slot of the week plan): 201, or 200 with
// the workout already in progress of that slot.
func (h *Handler) StartWorkout(ctx context.Context, req api.StartWorkoutRequestObject) (api.StartWorkoutResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	wo, created, err := h.workouts.Start(ctx, userID, req.Body.WeekSessionId)
	if err != nil {
		return nil, err
	}
	if created {
		return api.StartWorkout201JSONResponse(workoutToAPI(wo)), nil
	}
	return api.StartWorkout200JSONResponse(workoutToAPI(wo)), nil
}

// GetWorkout returns one of the user's workouts with its sets.
func (h *Handler) GetWorkout(ctx context.Context, req api.GetWorkoutRequestObject) (api.GetWorkoutResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	wo, err := h.workouts.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return api.GetWorkout200JSONResponse(workoutToAPI(wo)), nil
}

// FinishWorkout completes a workout with notes (or only updates the notes of a finished one),
// optionally dated to when it was done.
func (h *Handler) FinishWorkout(ctx context.Context, req api.FinishWorkoutRequestObject) (api.FinishWorkoutResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	wo, err := h.workouts.Finish(ctx, userID, req.Id, req.Body.Notes, req.Body.CompletedAt)
	if err != nil {
		return nil, err
	}
	return api.FinishWorkout200JSONResponse(workoutToAPI(wo)), nil
}

// DeleteWorkout discards a workout in progress or removes a finished one.
func (h *Handler) DeleteWorkout(ctx context.Context, req api.DeleteWorkoutRequestObject) (api.DeleteWorkoutResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.workouts.Delete(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return api.DeleteWorkout204Response{}, nil
}

// LogSet stores a set as it was done, or corrects one logged before.
func (h *Handler) LogSet(ctx context.Context, req api.LogSetRequestObject) (api.LogSetResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	set, err := h.workouts.LogSet(ctx, userID, req.Id, workout.SetInput{
		ExerciseID: b.ExerciseId, BlockPosition: b.BlockPosition, ExercisePosition: b.ExercisePosition, SetNumber: b.SetNumber,
		RepsValue: b.RepsValue, RepsUnit: b.RepsUnit, PerSide: b.PerSide, LoadValue: b.LoadValue, LoadUnit: b.LoadUnit,
	})
	if err != nil {
		return nil, err
	}
	return api.LogSet200JSONResponse{
		Id: set.ID, WorkoutId: set.WorkoutID, ExerciseId: set.ExerciseID,
		BlockPosition: set.BlockPosition, ExercisePosition: set.ExercisePosition, SetNumber: set.SetNumber,
		RepsValue: set.RepsValue, RepsUnit: set.RepsUnit, PerSide: set.PerSide, LoadValue: set.LoadValue, LoadUnit: set.LoadUnit,
		CompletedAt: timestamp(set.CompletedAt),
	}, nil
}

// DeleteSet removes a set from a workout.
func (h *Handler) DeleteSet(ctx context.Context, req api.DeleteSetRequestObject) (api.DeleteSetResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.workouts.DeleteSet(ctx, userID, req.Id, req.SetID); err != nil {
		return nil, err
	}
	return api.DeleteSet204Response{}, nil
}

// LogResult stores the score of a WOD block as it was done, or corrects one logged before.
func (h *Handler) LogResult(ctx context.Context, req api.LogResultRequestObject) (api.LogResultResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	result, err := h.workouts.LogResult(ctx, userID, req.Id, workout.ResultInput{
		BlockPosition: b.BlockPosition, ScoreType: string(b.ScoreType), TimeSeconds: b.TimeSeconds,
		Rounds: b.Rounds, Reps: b.Reps, Calories: b.Calories, Rx: b.Rx, Notes: b.Notes,
	})
	if err != nil {
		return nil, err
	}
	return api.LogResult200JSONResponse(workoutResultToAPI(result)), nil
}

// DeleteResult removes the score of a WOD block from a workout.
func (h *Handler) DeleteResult(ctx context.Context, req api.DeleteResultRequestObject) (api.DeleteResultResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.workouts.DeleteResult(ctx, userID, req.Id, req.ResultID); err != nil {
		return nil, err
	}
	return api.DeleteResult204Response{}, nil
}

func workoutToAPI(w workout.Workout) api.Workout {
	return api.Workout{
		Id: w.ID, TrainingId: w.TrainingID, WeekSessionId: w.WeekSessionID, SessionId: w.SessionID, SessionName: w.SessionName,
		StartedAt: timestamp(w.StartedAt), CompletedAt: optionalTimestamp(w.CompletedAt), Notes: w.Notes,
		Sets: mapSlice(w.Sets, func(s workoutrepos.ListWorkoutSetsRow) api.WorkoutSet {
			return api.WorkoutSet{
				Id: s.ID, ExerciseId: s.ExerciseID, ExerciseName: s.ExerciseName,
				BlockPosition: s.BlockPosition, ExercisePosition: s.ExercisePosition, SetNumber: s.SetNumber,
				RepsValue: s.RepsValue, RepsUnit: s.RepsUnit, RepsUnitName: s.RepsUnitName, PerSide: s.PerSide,
				LoadValue: s.LoadValue, LoadUnit: s.LoadUnit, LoadUnitName: s.LoadUnitName, CompletedAt: timestamp(s.CompletedAt),
			}
		}),
		Results: mapSlice(w.Results, workoutResultToAPI),
	}
}

func workoutResultToAPI(r workoutrepos.WorkoutResult) api.WorkoutResult {
	return api.WorkoutResult{
		Id: r.ID, WorkoutId: r.WorkoutID, BlockPosition: r.BlockPosition, ScoreType: api.ScoreType(r.ScoreType),
		TimeSeconds: r.TimeSeconds, Rounds: r.Rounds, Reps: r.Reps, Calories: r.Calories, Rx: r.Rx, Notes: r.Notes,
		CompletedAt: timestamp(r.CompletedAt),
	}
}

// workoutSummaryToAPI maps a finished workout as listed; completed_at is always set.
func workoutSummaryToAPI(w workoutrepos.ListWorkoutsRow) api.WorkoutSummary {
	return api.WorkoutSummary{
		Id: w.ID, TrainingId: w.TrainingID, WeekSessionId: w.WeekSessionID, SessionId: w.SessionID, SessionName: w.SessionName,
		StartedAt: timestamp(w.StartedAt), CompletedAt: timestamp(w.CompletedAt), Notes: w.Notes, SetCount: w.SetCount,
	}
}

func activeTrainingToAPI(t workoutrepos.ListActiveTrainingsRow) api.ActiveTraining {
	return api.ActiveTraining{
		Id: t.ID, Name: t.Name, StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks,
		CycleId: t.CycleID, CycleName: t.CycleName,
	}
}
