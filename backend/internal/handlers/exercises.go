package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service/exercise"
	exerciserepos "github.com/erdira/homegym/backend/internal/service/exercise/repos"
	"github.com/erdira/homegym/backend/internal/service/feedback"
	feedbackrepos "github.com/erdira/homegym/backend/internal/service/feedback/repos"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	lookuprepos "github.com/erdira/homegym/backend/internal/service/lookup/repos"
	"github.com/erdira/homegym/backend/internal/service/maxes"
	maxesrepos "github.com/erdira/homegym/backend/internal/service/maxes/repos"
)

// ListExercises returns the user's exercises with the names of their links.
func (h *Handler) ListExercises(ctx context.Context, _ api.ListExercisesRequestObject) (api.ListExercisesResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.exercises.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make(api.ListExercises200JSONResponse, len(rows))
	for i, row := range rows {
		if out[i], err = exerciseToAPI(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// GetExercise returns one of the user's exercises with the ids of its links, for filling the
// exercise form.
func (h *Handler) GetExercise(ctx context.Context, req api.GetExerciseRequestObject) (api.GetExerciseResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.exercises.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return api.GetExercise200JSONResponse{
		Id: e.ID, Name: e.Name, Description: e.Description, Type: api.ExerciseType(e.Type),
		VideoUrls: e.VideoUrls, JointIds: e.JointIds, MuscleGroupIds: e.MuscleGroupIds, FocusIds: e.FocusIds,
	}, nil
}

func (h *Handler) CreateExercise(ctx context.Context, req api.CreateExerciseRequestObject) (api.CreateExerciseResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.exercises.Create(ctx, userID, exerciseInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateExercise201JSONResponse(exerciseSavedToAPI(exerciserepos.UpdateExerciseRow(e))), nil
}

func (h *Handler) UpdateExercise(ctx context.Context, req api.UpdateExerciseRequestObject) (api.UpdateExerciseResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.exercises.Update(ctx, userID, req.Id, exerciseInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateExercise200JSONResponse(exerciseSavedToAPI(e)), nil
}

// RateExercise stores the user's rating (1-5) and notes for one of their exercises, replacing
// any earlier one.
func (h *Handler) RateExercise(ctx context.Context, req api.RateExerciseRequestObject) (api.RateExerciseResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	f, err := h.feedback.Rate(ctx, userID, req.Id, feedback.RatingInput{Rating: req.Body.Rating, Notes: req.Body.Notes})
	if err != nil {
		return nil, err
	}
	return api.RateExercise200JSONResponse{
		Id: f.ID, UserId: f.UserID, ExerciseId: f.ExerciseID, Rating: f.Rating, Notes: f.Notes,
		UpdatedAt: timestamp(f.UpdatedAt), DeletedAt: optionalTimestamp(f.DeletedAt),
	}, nil
}

// ListExerciseRatings returns how the user rated their exercises, for the exercise picker.
func (h *Handler) ListExerciseRatings(ctx context.Context, _ api.ListExerciseRatingsRequestObject) (api.ListExerciseRatingsResponseObject, error) {
	rows, err := listOwn(ctx, h.feedback.Ratings, func(r feedbackrepos.ListExerciseRatingsRow) api.ExerciseRating {
		return api.ExerciseRating{ExerciseId: r.ExerciseID, Rating: r.Rating, Notes: r.Notes, UpdatedAt: timestamp(r.UpdatedAt)}
	})
	return api.ListExerciseRatings200JSONResponse(rows), err
}

// AddExerciseMax adds a 1RM to the history of one of the user's exercises; the newest is the
// current max.
func (h *Handler) AddExerciseMax(ctx context.Context, req api.AddExerciseMaxRequestObject) (api.AddExerciseMaxResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	b := req.Body
	m, err := h.maxes.Add(ctx, userID, req.Id, maxes.MaxInput{
		LoadValue: b.LoadValue, LoadUnit: b.LoadUnit, AchievedOn: b.AchievedOn, WorkoutSetID: b.WorkoutSetId, Notes: b.Notes,
	})
	if err != nil {
		return nil, err
	}
	return api.AddExerciseMax201JSONResponse(exerciseMaxToAPI(maxesrepos.ListExerciseMaxesRow(m))), nil
}

// ListExerciseMaxes returns the user's 1RM history of all their exercises, newest first.
func (h *Handler) ListExerciseMaxes(ctx context.Context, _ api.ListExerciseMaxesRequestObject) (api.ListExerciseMaxesResponseObject, error) {
	rows, err := listOwn(ctx, h.maxes.List, exerciseMaxToAPI)
	return api.ListExerciseMaxes200JSONResponse(rows), err
}

// DeleteExerciseMax removes an entry from the user's 1RM history.
func (h *Handler) DeleteExerciseMax(ctx context.Context, req api.DeleteExerciseMaxRequestObject) (api.DeleteExerciseMaxResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.maxes.Delete(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return api.DeleteExerciseMax204Response{}, nil
}

// ExerciseStats lists the exercises the user has done in workouts, most often first, with
// their rating of each.
func (h *Handler) ExerciseStats(ctx context.Context, _ api.ExerciseStatsRequestObject) (api.ExerciseStatsResponseObject, error) {
	rows, err := listOwn(ctx, h.feedback.Stats, func(s feedbackrepos.GetExerciseStatsRow) api.ExerciseStat {
		return api.ExerciseStat{
			ExerciseId: s.ExerciseID, Name: s.Name, Joints: s.Joints, MuscleGroups: s.MuscleGroups,
			TimesPerformed: s.TimesPerformed, Rating: s.Rating,
		}
	})
	return api.ExerciseStats200JSONResponse(rows), err
}

func (h *Handler) ListJoints(ctx context.Context, _ api.ListJointsRequestObject) (api.ListJointsResponseObject, error) {
	rows, err := list(ctx, h.lookups.Joints, func(j lookuprepos.ListJointsRow) api.Option {
		return api.Option{Id: j.ID, Name: j.Name}
	})
	return api.ListJoints200JSONResponse(rows), err
}

func (h *Handler) ListMuscleGroups(ctx context.Context, _ api.ListMuscleGroupsRequestObject) (api.ListMuscleGroupsResponseObject, error) {
	rows, err := list(ctx, h.lookups.MuscleGroups, func(m lookuprepos.ListMuscleGroupsRow) api.Option {
		return api.Option{Id: m.ID, Name: m.Name}
	})
	return api.ListMuscleGroups200JSONResponse(rows), err
}

func (h *Handler) ListFocuses(ctx context.Context, _ api.ListFocusesRequestObject) (api.ListFocusesResponseObject, error) {
	rows, err := list(ctx, h.lookups.Focuses, func(f lookuprepos.ListFocusesRow) api.Focus {
		return api.Focus{Id: f.ID, Name: f.Name, Alias: f.Alias}
	})
	return api.ListFocuses200JSONResponse(rows), err
}

func (h *Handler) CreateFocus(ctx context.Context, req api.CreateFocusRequestObject) (api.CreateFocusResponseObject, error) {
	f, err := h.lookups.CreateFocus(ctx, lookup.FocusInput{Name: req.Body.Name, Alias: req.Body.Alias})
	if err != nil {
		return nil, err
	}
	return api.CreateFocus201JSONResponse{Id: f.ID, Name: f.Name, Alias: f.Alias}, nil
}

// exerciseToAPI maps a row of the exercise list; its focuses come as JSON from the query.
func exerciseToAPI(e exerciserepos.ListExercisesRow) (api.Exercise, error) {
	out := api.Exercise{
		Id: e.ID, Name: e.Name, Description: e.Description, Type: api.ExerciseType(e.Type), VideoUrls: e.VideoUrls,
		CreatedAt: timestamp(e.CreatedAt), UpdatedAt: timestamp(e.UpdatedAt), Joints: e.Joints, MuscleGroups: e.MuscleGroups,
	}
	if err := json.Unmarshal(e.Focuses, &out.Focuses); err != nil {
		return api.Exercise{}, fmt.Errorf("decode focuses of exercise %s: %w", e.ID, err)
	}
	return out, nil
}

// exerciseSavedToAPI maps a created or updated exercise (CreateExerciseRow converts to it).
func exerciseSavedToAPI(e exerciserepos.UpdateExerciseRow) api.ExerciseSaved {
	return api.ExerciseSaved{
		Id: e.ID, Name: e.Name, Description: e.Description, Type: api.ExerciseType(e.Type), VideoUrls: e.VideoUrls,
		CreatedAt: timestamp(e.CreatedAt), UpdatedAt: timestamp(e.UpdatedAt),
	}
}

func exerciseInputFromAPI(in api.ExerciseInput) exercise.Input {
	return exercise.Input{
		Name: in.Name, Type: string(in.Type), Description: in.Description, VideoURLs: in.VideoUrls,
		JointIDs: in.JointIds, MuscleGroupIDs: in.MuscleGroupIds, FocusIDs: in.FocusIds,
	}
}

// exerciseMaxToAPI maps a 1RM entry (CreateExerciseMaxRow converts to the list row).
func exerciseMaxToAPI(m maxesrepos.ListExerciseMaxesRow) api.ExerciseMax {
	return api.ExerciseMax{
		Id: m.ID, ExerciseId: m.ExerciseID, LoadValue: m.LoadValue, LoadUnit: m.LoadUnit, LoadUnitName: m.LoadUnitName,
		AchievedOn: date(m.AchievedOn), WorkoutSetId: m.WorkoutSetID, Notes: m.Notes, CreatedAt: timestamp(m.CreatedAt),
	}
}
