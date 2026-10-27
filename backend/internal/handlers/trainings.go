package handlers

import (
	"context"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// ListTrainings returns the user's trainings with their size, for the overview.
func (h *Handler) ListTrainings(ctx context.Context, _ api.ListTrainingsRequestObject) (api.ListTrainingsResponseObject, error) {
	rows, err := listOwn(ctx, h.trainings.List, func(t trainingrepos.ListTrainingsRow) api.Training {
		return api.Training{
			Id: t.ID, CycleId: t.CycleID, CycleName: t.CycleName, Name: t.Name, Description: t.Description, Active: t.Active,
			StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks, UpdatedAt: timestamp(t.UpdatedAt),
			SessionCount: t.SessionCount, PlannedCount: t.PlannedCount,
		}
	})
	return api.ListTrainings200JSONResponse(rows), err
}

// GetTraining returns one of the user's trainings with its sessions and schedule.
func (h *Handler) GetTraining(ctx context.Context, req api.GetTrainingRequestObject) (api.GetTrainingResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.trainings.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	saved := trainingSavedToAPI(trainingrepos.CreateTrainingRow(t.GetTrainingRow))
	return api.GetTraining200JSONResponse{
		Id: saved.Id, CycleId: saved.CycleId, Name: saved.Name, Description: saved.Description, Active: saved.Active,
		StartDate: saved.StartDate, EndDate: saved.EndDate, Weeks: saved.Weeks, CreatedAt: saved.CreatedAt, UpdatedAt: saved.UpdatedAt,
		Sessions: mapSlice(t.Sessions, sessionToAPI),
		Schedule: mapSlice(t.Schedule, func(week []training.Slot) []api.Slot {
			return mapSlice(week, func(s training.Slot) api.Slot { return api.Slot{Id: s.ID, SessionId: s.SessionID} })
		}),
	}, nil
}

// CreateTraining saves a new training with its whole tree.
func (h *Handler) CreateTraining(ctx context.Context, req api.CreateTrainingRequestObject) (api.CreateTrainingResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.trainings.Create(ctx, userID, trainingInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateTraining201JSONResponse(trainingSavedToAPI(t)), nil
}

// UpdateTraining replaces a training's fields and tree.
func (h *Handler) UpdateTraining(ctx context.Context, req api.UpdateTrainingRequestObject) (api.UpdateTrainingResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.trainings.Update(ctx, userID, req.Id, trainingInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateTraining200JSONResponse(trainingSavedToAPI(trainingrepos.CreateTrainingRow(t))), nil
}

// DuplicateTraining copies a training as the next phase, inactive.
func (h *Handler) DuplicateTraining(ctx context.Context, req api.DuplicateTrainingRequestObject) (api.DuplicateTrainingResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.trainings.Duplicate(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return api.DuplicateTraining201JSONResponse(trainingSavedToAPI(t)), nil
}

// DeleteTraining soft-deletes a training.
func (h *Handler) DeleteTraining(ctx context.Context, req api.DeleteTrainingRequestObject) (api.DeleteTrainingResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.trainings.Delete(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return api.DeleteTraining204Response{}, nil
}

// trainingSavedToAPI maps a training's own fields (the Get and Update rows convert to it).
func trainingSavedToAPI(t trainingrepos.CreateTrainingRow) api.TrainingSaved {
	return api.TrainingSaved{
		Id: t.ID, CycleId: t.CycleID, Name: t.Name, Description: t.Description, Active: t.Active,
		StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks,
		CreatedAt: timestamp(t.CreatedAt), UpdatedAt: timestamp(t.UpdatedAt),
	}
}

// The tree of a training (and of a library session): sessions → blocks → exercises.

func sessionToAPI(s training.Session) api.TrainingSession {
	return api.TrainingSession{Id: s.ID, Name: s.Name, Notes: s.Notes, Blocks: mapSlice(s.Blocks, blockToAPI)}
}

func blockToAPI(b training.Block) api.TrainingBlock {
	return api.TrainingBlock{
		Id: b.ID, Kind: b.Kind, Name: b.Name, Sets: b.Sets,
		RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, RepsUnitName: b.RepsUnitName, PerSide: b.PerSide,
		LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit, LoadUnitName: b.LoadUnitName,
		ScoreType: (*api.ScoreType)(b.ScoreType), TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds,
		Notes: b.Notes, Exercises: mapSlice(b.Exercises, trainingExerciseToAPI),
	}
}

func trainingExerciseToAPI(e training.Exercise) api.TrainingExercise {
	return api.TrainingExercise{
		ExerciseId: e.ExerciseID, Name: e.Name, Type: api.ExerciseType(e.Type), Sets: e.Sets,
		RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, RepsUnitName: e.RepsUnitName, PerSide: e.PerSide,
		LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit, LoadUnitName: e.LoadUnitName,
		Tempo: e.Tempo, Notes: e.Notes,
	}
}

func trainingInputFromAPI(in api.TrainingInput) training.Input {
	return training.Input{
		CycleID: in.CycleId, Name: in.Name, Description: in.Description, Active: in.Active,
		StartDate: in.StartDate, Weeks: in.Weeks,
		Sessions: mapSlice(in.Sessions, sessionInputFromAPI),
		Schedule: mapSlice(in.Schedule, func(week []api.SlotInput) []training.SlotInput {
			return mapSlice(week, func(s api.SlotInput) training.SlotInput { return training.SlotInput{ID: s.Id, Session: s.Session} })
		}),
	}
}

func sessionInputFromAPI(s api.TrainingSessionInput) training.SessionInput {
	return training.SessionInput{ID: s.Id, Name: s.Name, Notes: s.Notes, Blocks: mapSlice(s.Blocks, blockInputFromAPI)}
}

func blockInputFromAPI(b api.TrainingBlockInput) training.BlockInput {
	return training.BlockInput{
		Kind: b.Kind, Name: b.Name, Sets: b.Sets,
		Amount: training.Amount{
			RepsValue: b.RepsValue, RepsMax: b.RepsMax, RepsUnit: b.RepsUnit, PerSide: b.PerSide,
			LoadValue: b.LoadValue, LoadMax: b.LoadMax, LoadUnit: b.LoadUnit,
		},
		Score:     training.Score{ScoreType: (*string)(b.ScoreType), TimeSeconds: b.TimeSeconds, IntervalSeconds: b.IntervalSeconds},
		Notes:     b.Notes,
		Exercises: mapSlice(b.Exercises, trainingExerciseInputFromAPI),
	}
}

func trainingExerciseInputFromAPI(e api.TrainingExerciseInput) training.ExerciseInput {
	return training.ExerciseInput{
		ExerciseID: e.ExerciseId, Sets: e.Sets,
		Amount: training.Amount{
			RepsValue: e.RepsValue, RepsMax: e.RepsMax, RepsUnit: e.RepsUnit, PerSide: e.PerSide,
			LoadValue: e.LoadValue, LoadMax: e.LoadMax, LoadUnit: e.LoadUnit,
		},
		Tempo: e.Tempo, Notes: e.Notes,
	}
}
