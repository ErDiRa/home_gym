package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// ExerciseNoteInput is a note on an exercise as it was done in a workout, at its place in the
// session (from 1) like its sets: noting it again replaces the note.
type ExerciseNoteInput struct {
	ExerciseID       uuid.UUID
	BlockPosition    int32
	ExercisePosition int32
	Notes            string
}

// ExerciseNote is a note on an exercise of a workout as stored.
type ExerciseNote struct {
	ID               uuid.UUID
	ExerciseID       uuid.UUID
	BlockPosition    int32
	ExercisePosition int32
	Notes            string
	UpdatedAt        time.Time
}

// NoteExercise stores a note on an exercise of one of the user's workouts, or replaces the one it
// had; like sets, also after finishing. The exercise must be one of the user's live exercises.
func (s *Service) NoteExercise(ctx context.Context, userID, workoutID uuid.UUID, in ExerciseNoteInput) (ExerciseNote, error) {
	in.Notes = strings.TrimSpace(in.Notes)
	switch {
	case in.ExerciseID == uuid.Nil:
		return ExerciseNote{}, errs.Invalid("exercise_id is required")
	case in.BlockPosition < 1 || in.ExercisePosition < 1:
		return ExerciseNote{}, errs.Invalid("block_position and exercise_position must be 1 or more")
	case in.Notes == "":
		return ExerciseNote{}, errs.Invalid("enter a note")
	case utf8.RuneCountInString(in.Notes) > 1000:
		return ExerciseNote{}, errs.Invalid("the note can have at most 1000 characters")
	}
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return ExerciseNote{}, err
	}
	exercises, err := s.catalog.Exercises(ctx, userID, []uuid.UUID{in.ExerciseID})
	if err != nil {
		return ExerciseNote{}, err
	}
	if e, ok := exercises[in.ExerciseID]; !ok || e.Deleted {
		return ExerciseNote{}, errs.Invalid("unknown exercise")
	}
	return s.workouts.NoteExercise(ctx, workoutID, in)
}

// DeleteExerciseNote removes a note from an exercise of one of the user's workouts.
func (s *Service) DeleteExerciseNote(ctx context.Context, userID, workoutID, noteID uuid.UUID) error {
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return err
	}
	if err := s.workouts.DeleteExerciseNote(ctx, workoutID, noteID); errors.Is(err, ErrNotFound) {
		return errs.NotFound("note not found", err)
	} else if err != nil {
		return err
	}
	return nil
}
