// Package exercise manages a user's exercise library, which their trainings are built from.
// Every usecase acts on the given user's exercises only; another user's look missing.
package exercise

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/exercise/repos"
)

// Queries is the part of repos.Queries this service uses; *repos.Queries satisfies it.
type Queries interface {
	ListExercises(ctx context.Context, userID uuid.UUID) ([]repos.ListExercisesRow, error)
	GetExercise(ctx context.Context, arg repos.GetExerciseParams) (repos.GetExerciseRow, error)
	CreateExercise(ctx context.Context, arg repos.CreateExerciseParams) (repos.CreateExerciseRow, error)
	UpdateExercise(ctx context.Context, arg repos.UpdateExerciseParams) (repos.UpdateExerciseRow, error)
}

// Service implements the exercise usecases.
type Service struct {
	q Queries
}

// New builds the service on Postgres.
func New(pool *pgxpool.Pool) *Service { return &Service{q: repos.New(pool)} }

// Input is an exercise for Create and Update.
type Input struct {
	Name           string      `json:"name"`
	JointIDs       []uuid.UUID `json:"joint_ids"`
	MuscleGroupIDs []uuid.UUID `json:"muscle_group_ids"`
	FocusIDs       []uuid.UUID `json:"focus_ids"`
	Description    string      `json:"description"`
	VideoURLs      []string    `json:"video_urls"` // in display order
	Type           string      `json:"type"`
}

// validate checks in, in place: ids are deduplicated (a repeated id would violate the link
// tables' primary keys) and only non-blank video links are kept.
func validate(in *Input) error {
	in.JointIDs, in.MuscleGroupIDs, in.FocusIDs = unique(in.JointIDs), unique(in.MuscleGroupIDs), unique(in.FocusIDs)
	if in.Name == "" || len(in.JointIDs) == 0 || len(in.MuscleGroupIDs) == 0 {
		return service.Invalid("name, joint_ids and muscle_group_ids are required")
	}
	if in.Type != "mobility" && in.Type != "strength" {
		return service.Invalid("type must be mobility or strength")
	}
	// Empty inputs are dropped, so the form can send the blank row it shows for adding a link.
	videos := make([]string, 0, len(in.VideoURLs))
	for _, v := range in.VideoURLs {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		if u, err := url.Parse(v); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || tooLong(v, 500) {
			return service.Invalid("video links must be http(s) URLs (max 500 characters)")
		}
		videos = append(videos, v)
	}
	if len(videos) > 10 {
		return service.Invalid("an exercise can have at most 10 video links")
	}
	in.VideoURLs = videos
	return nil
}

// unique returns ids without duplicates, keeping the first occurrence of each.
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

// mapError turns a failed query into the error the caller sees: no row means the exercise is
// missing or deleted, a foreign key violation means a linked id doesn't exist.
func mapError(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return service.NotFound("exercise not found", err)
	case isForeignKeyViolation(err):
		return service.InvalidCause("unknown joint, muscle group or focus", err)
	default:
		return err
	}
}

// tooLong reports whether s has more than max characters (runes, not bytes).
func tooLong(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// isForeignKeyViolation reports whether err is a Postgres foreign_key_violation.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
