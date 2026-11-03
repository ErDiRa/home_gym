package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// maxTemplateExercises bounds the exercises of a saved block (the kind may allow fewer).
const maxTemplateExercises = 20

// BlockTemplateInput is a saved block as the user sends it: a name, the kind of block it makes and
// its exercises in order.
type BlockTemplateInput struct {
	Name        string
	Kind        string
	ExerciseIDs []uuid.UUID
}

// CreateBlockTemplate validates a new saved block and saves it.
func (s *Service) CreateBlockTemplate(ctx context.Context, userID uuid.UUID, in BlockTemplateInput) (uuid.UUID, error) {
	if err := s.validateBlockTemplate(ctx, userID, &in); err != nil {
		return uuid.Nil, err
	}
	id, err := s.library.CreateBlockTemplate(ctx, userID, in)
	return id, blockTemplateError(err)
}

// UpdateBlockTemplate validates a saved block and replaces it. Sessions it was added to keep their
// own blocks.
func (s *Service) UpdateBlockTemplate(ctx context.Context, userID, id uuid.UUID, in BlockTemplateInput) error {
	if err := s.validateBlockTemplate(ctx, userID, &in); err != nil {
		return err
	}
	return blockTemplateError(s.library.UpdateBlockTemplate(ctx, userID, id, in))
}

// DeleteBlockTemplate soft-deletes one of the user's saved blocks.
func (s *Service) DeleteBlockTemplate(ctx context.Context, userID, id uuid.UUID) error {
	return blockTemplateError(s.library.DeleteBlockTemplate(ctx, userID, id))
}

// validateBlockTemplate checks a saved block in place: a name, a known kind with as many exercises
// as it takes, each one of the user's live exercises.
func (s *Service) validateBlockTemplate(ctx context.Context, userID uuid.UUID, in *BlockTemplateInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || tooLong(in.Name, 100) {
		return errs.Invalid("a saved block needs a name (max 100 characters)")
	}
	lk, err := s.lookups(ctx)
	if err != nil {
		return err
	}
	kind, ok := lk.kinds[in.Kind]
	if !ok {
		return errs.Invalid("unknown block kind")
	}
	n := int32(len(in.ExerciseIDs))
	if n == 0 || n > maxTemplateExercises {
		return errs.Invalid("a saved block takes 1 to 20 exercises")
	}
	if n < kind.MinExercises || (kind.MaxExercises != nil && n > *kind.MaxExercises) {
		return errs.Invalid(exerciseCountRule(kind))
	}
	exercises, err := s.catalog.Exercises(ctx, userID, in.ExerciseIDs)
	if err != nil {
		return err
	}
	for _, id := range in.ExerciseIDs {
		if e, ok := exercises[id]; !ok || e.Deleted {
			return errs.Invalid("unknown exercise")
		}
	}
	return nil
}

// blockTemplateError maps the repository's errors: a missing row is a missing saved block.
func blockTemplateError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("saved block not found", err)
	case errors.Is(err, ErrDuplicate):
		return errs.Conflict("a saved block with this name exists", err)
	default:
		return trainingError(err)
	}
}
