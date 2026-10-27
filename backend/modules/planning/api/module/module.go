// Package module implements the planning module's contract (client.Planning). Its reads are plain
// data retrieval, like the read model's.
package module

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/modules/planning/adapters/db/dbmodels"
	"github.com/erdira/homegym/backend/modules/planning/api/module/client"
)

type Planning struct {
	q *dbmodels.Queries
}

var _ client.Planning = (*Planning)(nil)

func New(pool *pgxpool.Pool) *Planning {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &Planning{q: dbmodels.New(pool)}
}

func (p *Planning) ActiveTrainings(ctx context.Context, userID uuid.UUID) ([]client.ActiveTraining, error) {
	rows, err := p.q.ListActiveTrainings(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]client.ActiveTraining, len(rows))
	for i, t := range rows {
		out[i] = client.ActiveTraining(t)
	}
	return out, nil
}

func (p *Planning) WeekPlan(ctx context.Context, userID, trainingID uuid.UUID, week int32) ([]client.PlannedSlot, error) {
	rows, err := p.q.ListWeekPlan(ctx, dbmodels.ListWeekPlanParams{UserID: userID, TrainingID: trainingID, Week: week})
	if err != nil {
		return nil, err
	}
	out := make([]client.PlannedSlot, len(rows))
	for i, s := range rows {
		out[i] = client.PlannedSlot{ID: s.ID, SessionID: s.SessionID, SessionName: s.Name}
	}
	return out, nil
}

func (p *Planning) Slot(ctx context.Context, userID, slotID uuid.UUID) (client.Slot, error) {
	s, err := p.q.GetWeekSlot(ctx, dbmodels.GetWeekSlotParams{ID: slotID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return client.Slot{}, errors.Join(client.ErrNotFound, err)
	}
	if err != nil {
		return client.Slot{}, err
	}
	return client.Slot(s), nil
}

func (p *Planning) Cycle(ctx context.Context, userID, cycleID uuid.UUID) (client.Cycle, error) {
	c, err := p.q.GetCycleOverview(ctx, dbmodels.GetCycleOverviewParams{ID: cycleID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return client.Cycle{}, errors.Join(client.ErrNotFound, err)
	}
	if err != nil {
		return client.Cycle{}, err
	}
	goals, err := p.q.ListCycleGoalStatuses(ctx, cycleID)
	if err != nil {
		return client.Cycle{}, err
	}
	out := client.Cycle{ID: c.ID, Name: c.Name, StartDate: c.StartDate, EndDate: c.EndDate, Weeks: c.Weeks, Goals: make([]client.Goal, len(goals))}
	for i, g := range goals {
		out.Goals[i] = client.Goal(g)
	}
	return out, nil
}
