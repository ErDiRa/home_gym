package app

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

func TestBlockTemplates(t *testing.T) {
	ctx := context.Background()
	s, repo, cat := newTestService()
	a, b := cat.exercise(), cat.exercise()
	if _, err := s.CreateBlockTemplate(ctx, uuid.New(), BlockTemplateInput{Name: " Clean complex ", Kind: "superset", ExerciseIDs: []uuid.UUID{a, b}}); err != nil {
		t.Fatal(err)
	}
	if got := repo.templates[0]; got.Name != "Clean complex" || !slices.Equal(got.ExerciseIDs, []uuid.UUID{a, b}) {
		t.Errorf("saved = %+v", got)
	}

	tests := []struct {
		name string
		in   BlockTemplateInput
		msg  string
	}{
		{"no name", BlockTemplateInput{Name: " ", Kind: "single", ExerciseIDs: []uuid.UUID{a}}, "a saved block needs a name (max 100 characters)"},
		{"unknown kind", BlockTemplateInput{Name: "X", Kind: "circuit", ExerciseIDs: []uuid.UUID{a}}, "unknown block kind"},
		{"no exercises", BlockTemplateInput{Name: "X", Kind: "single"}, "a saved block takes 1 to 20 exercises"},
		{"too few for the kind", BlockTemplateInput{Name: "X", Kind: "superset", ExerciseIDs: []uuid.UUID{a}}, "a Superset block needs at least 2 exercises"},
		{"unknown exercise", BlockTemplateInput{Name: "X", Kind: "single", ExerciseIDs: []uuid.UUID{a, uuid.New()}}, "unknown exercise"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.CreateBlockTemplate(ctx, uuid.New(), tt.in)
			wantError(t, err, errs.KindInvalid, tt.msg)
		})
	}

	valid := BlockTemplateInput{Name: "X", Kind: "single", ExerciseIDs: []uuid.UUID{a}}
	repo.err = ErrDuplicate
	_, err := s.CreateBlockTemplate(ctx, uuid.New(), valid)
	wantError(t, err, errs.KindConflict, "a saved block with this name exists")
	repo.err = ErrNotFound
	wantError(t, s.UpdateBlockTemplate(ctx, uuid.New(), uuid.New(), valid), errs.KindNotFound, "saved block not found")
	wantError(t, s.DeleteBlockTemplate(ctx, uuid.New(), uuid.New()), errs.KindNotFound, "saved block not found")
}
