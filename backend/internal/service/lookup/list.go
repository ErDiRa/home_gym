package lookup

import (
	"context"

	"github.com/erdira/homegym/backend/internal/service/lookup/repos"
)

func (s *Service) Joints(ctx context.Context) ([]repos.ListJointsRow, error) {
	return s.q.ListJoints(ctx)
}

func (s *Service) MuscleGroups(ctx context.Context) ([]repos.ListMuscleGroupsRow, error) {
	return s.q.ListMuscleGroups(ctx)
}

func (s *Service) Focuses(ctx context.Context) ([]repos.ListFocusesRow, error) {
	return s.q.ListFocuses(ctx)
}

func (s *Service) BlockKinds(ctx context.Context) ([]repos.ListBlockKindsRow, error) {
	return s.q.ListBlockKinds(ctx)
}

func (s *Service) RepUnits(ctx context.Context) ([]repos.ListRepUnitsRow, error) {
	return s.q.ListRepUnits(ctx)
}

func (s *Service) LoadUnits(ctx context.Context) ([]repos.ListLoadUnitsRow, error) {
	return s.q.ListLoadUnits(ctx)
}
