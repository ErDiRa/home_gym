package handlers

import (
	"context"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/cycle"
	cyclerepos "github.com/erdira/homegym/backend/internal/service/cycle/repos"
	"github.com/erdira/homegym/backend/internal/service/training"
)

// ListCycles returns the user's cycles with their goal and training counts.
func (h *Handler) ListCycles(ctx context.Context, _ api.ListCyclesRequestObject) (api.ListCyclesResponseObject, error) {
	rows, err := listOwn(ctx, h.cycles.List, func(c cyclerepos.ListCyclesRow) api.CycleSummary {
		return api.CycleSummary{
			Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
			Weeks: c.Weeks, UpdatedAt: timestamp(c.UpdatedAt), GoalCount: c.GoalCount, TrainingCount: c.TrainingCount,
		}
	})
	return api.ListCycles200JSONResponse(rows), err
}

// GetCycle returns one of the user's cycles with its goals and trainings.
func (h *Handler) GetCycle(ctx context.Context, req api.GetCycleRequestObject) (api.GetCycleResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.cycles.Get(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return api.GetCycle200JSONResponse(cycleToAPI(c)), nil
}

// CreateCycle saves a new cycle with its goals.
func (h *Handler) CreateCycle(ctx context.Context, req api.CreateCycleRequestObject) (api.CreateCycleResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.cycles.Create(ctx, userID, cycleInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateCycle201JSONResponse(cycleToAPI(c)), nil
}

// UpdateCycle replaces a cycle's fields and goals.
func (h *Handler) UpdateCycle(ctx context.Context, req api.UpdateCycleRequestObject) (api.UpdateCycleResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.cycles.Update(ctx, userID, req.Id, cycleInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateCycle200JSONResponse(cycleToAPI(c)), nil
}

// DeleteCycle soft-deletes a cycle. ?trainings=delete deletes its trainings too; by default
// (keep) they stay and just leave the cycle.
func (h *Handler) DeleteCycle(ctx context.Context, req api.DeleteCycleRequestObject) (api.DeleteCycleResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	var withTrainings bool
	switch value(req.Params.Trainings) {
	case "", api.Keep:
	case api.Delete:
		withTrainings = true
	default:
		return nil, service.Invalid("trainings must be keep or delete")
	}
	if err := h.cycles.Delete(ctx, userID, req.Id, withTrainings); err != nil {
		return nil, err
	}
	return api.DeleteCycle204Response{}, nil
}

// ImportCycle saves a whole cycle with its trainings at once, or nothing if any part is invalid
// (a usecase of the training service, which writes the trainings). Its body may be larger than
// others' (see limitBody).
func (h *Handler) ImportCycle(ctx context.Context, req api.ImportCycleRequestObject) (api.ImportCycleResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.trainings.ImportCycle(ctx, userID, training.CycleImportInput{
		Cycle:     cycleInputFromAPI(req.Body.Cycle),
		Trainings: mapSlice(req.Body.Trainings, trainingInputFromAPI),
	})
	if err != nil {
		return nil, err
	}
	return api.ImportCycle201JSONResponse{
		Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
		Weeks: c.Weeks, CreatedAt: timestamp(c.CreatedAt), UpdatedAt: timestamp(c.UpdatedAt),
		Trainings: mapSlice(c.Trainings, trainingSavedToAPI),
	}, nil
}

func cycleToAPI(c cycle.Cycle) api.Cycle {
	return api.Cycle{
		Id: c.ID, Name: c.Name, Description: c.Description, StartDate: date(c.StartDate), EndDate: date(c.EndDate),
		Weeks: c.Weeks, CreatedAt: timestamp(c.CreatedAt), UpdatedAt: timestamp(c.UpdatedAt),
		Goals: mapSlice(c.Goals, func(g cyclerepos.ListCycleGoalsRow) api.CycleGoal {
			return api.CycleGoal{
				Id: g.ID, Position: g.Position, Title: g.Title, Target: g.Target, Status: api.GoalStatus(g.Status), Notes: g.Notes,
			}
		}),
		Trainings: mapSlice(c.Trainings, func(t cyclerepos.ListCycleTrainingsRow) api.CycleTraining {
			return api.CycleTraining{
				Id: t.ID, Name: t.Name, Active: t.Active, StartDate: date(t.StartDate), EndDate: date(t.EndDate), Weeks: t.Weeks,
			}
		}),
	}
}

func cycleInputFromAPI(in api.CycleInput) cycle.Input {
	return cycle.Input{
		Name: in.Name, Description: in.Description, StartDate: in.StartDate, Weeks: in.Weeks,
		Goals: mapSlice(in.Goals, func(g api.CycleGoalInput) cycle.GoalInput {
			return cycle.GoalInput{ID: g.Id, Title: g.Title, Target: g.Target, Status: string(g.Status), Notes: g.Notes}
		}),
	}
}
