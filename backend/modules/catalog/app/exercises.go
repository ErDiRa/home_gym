package app

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// ExerciseInput is an exercise for Create and Update.
type ExerciseInput struct {
	Name           string
	Type           string
	Description    string
	VideoURLs      []string // in display order
	JointIDs       []uuid.UUID
	MuscleGroupIDs []uuid.UUID
	FocusIDs       []uuid.UUID
}

// CreateExercise adds an exercise to the user's library, with its links and videos.
func (s *Service) CreateExercise(ctx context.Context, userID uuid.UUID, in ExerciseInput) (Exercise, error) {
	if err := validateExercise(&in); err != nil {
		return Exercise{}, err
	}
	e, err := s.exercises.CreateExercise(ctx, userID, in)
	return e, exerciseError(err)
}

// UpdateExercise replaces one of the user's exercises: its fields, links and videos.
func (s *Service) UpdateExercise(ctx context.Context, userID, id uuid.UUID, in ExerciseInput) (Exercise, error) {
	if err := validateExercise(&in); err != nil {
		return Exercise{}, err
	}
	e, err := s.exercises.UpdateExercise(ctx, userID, id, in)
	return e, exerciseError(err)
}

// validateExercise checks in, in place: ids are deduplicated (a repeated id would violate the link
// tables' primary keys) and only non-blank video links are kept.
func validateExercise(in *ExerciseInput) error {
	in.JointIDs, in.MuscleGroupIDs, in.FocusIDs = unique(in.JointIDs), unique(in.MuscleGroupIDs), unique(in.FocusIDs)
	if in.Name == "" {
		return errs.Invalid("name is required")
	}
	if in.Type != "mobility" && in.Type != "strength" {
		return errs.Invalid("type must be mobility or strength")
	}
	// Empty inputs are dropped, so the form can send the blank row it shows for adding a link.
	videos := make([]string, 0, len(in.VideoURLs))
	for _, v := range in.VideoURLs {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		if u, err := url.Parse(v); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || tooLong(v, 500) {
			return errs.Invalid("video links must be http(s) URLs (max 500 characters)")
		}
		videos = append(videos, v)
	}
	if len(videos) > 10 {
		return errs.Invalid("an exercise can have at most 10 video links")
	}
	in.VideoURLs = videos
	return nil
}

// unique returns ids without duplicates, keeping the first occurrence of each; never nil.
func unique(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// exerciseError turns a repository's error into the one the caller sees.
func exerciseError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("exercise not found", err)
	case errors.Is(err, ErrUnknownReference):
		return errs.InvalidCause("unknown joint, muscle group or focus", err)
	default:
		return err
	}
}
