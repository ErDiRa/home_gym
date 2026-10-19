package training

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// LibrarySession is a session template of the user's library with its blocks and exercises.
type LibrarySession struct {
	repos.GetLibrarySessionRow
	Blocks []Block `json:"blocks"`
}

// ListSessions returns the user's session templates with their size, by name.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]repos.ListLibrarySessionsRow, error) {
	return s.q.ListLibrarySessions(ctx, userID)
}

// GetSession returns one of the user's session templates with its blocks and exercises.
func (s *Service) GetSession(ctx context.Context, userID, id uuid.UUID) (LibrarySession, error) {
	session, err := loadSession(ctx, s.q, userID, id)
	return session, sessionError(err)
}

// CreateSession validates a new session template and saves it with its blocks in one
// transaction.
func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, in SessionInput) (LibrarySession, error) {
	if err := s.validateTemplate(ctx, &in); err != nil {
		return LibrarySession{}, err
	}
	var session LibrarySession
	err := s.inTx(ctx, func(q Queries) error {
		id, err := q.InsertSession(ctx, repos.InsertSessionParams{UserID: userID, Name: in.Name, Notes: in.Notes})
		if err != nil {
			return err
		}
		if err := insertBlocks(ctx, q, userID, id, in.Blocks); err != nil {
			return err
		}
		session, err = loadSession(ctx, q, userID, id)
		return err
	})
	return session, sessionError(err)
}

// UpdateSession validates a session template and replaces its fields and blocks in one
// transaction. Trainings it was added to keep their own copies.
func (s *Service) UpdateSession(ctx context.Context, userID, id uuid.UUID, in SessionInput) (LibrarySession, error) {
	if err := s.validateTemplate(ctx, &in); err != nil {
		return LibrarySession{}, err
	}
	var session LibrarySession
	err := s.inTx(ctx, func(q Queries) error {
		n, err := q.UpdateLibrarySession(ctx, repos.UpdateLibrarySessionParams{ID: id, UserID: userID, Name: in.Name, Notes: in.Notes})
		if err != nil {
			return err
		}
		if n == 0 {
			return pgx.ErrNoRows
		}
		if err := q.DeleteSessionBlocks(ctx, id); err != nil {
			return err
		}
		if err := insertBlocks(ctx, q, userID, id, in.Blocks); err != nil {
			return err
		}
		session, err = loadSession(ctx, q, userID, id)
		return err
	})
	return session, sessionError(err)
}

// DeleteSession soft-deletes one of the user's session templates. Trainings it was added to
// keep their own copies.
func (s *Service) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.DeleteLibrarySession(ctx, repos.DeleteLibrarySessionParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("session not found", nil)
	}
	return nil
}

// validateTemplate trims and validates a session template against the lookups, in place.
func (s *Service) validateTemplate(ctx context.Context, in *SessionInput) error {
	lk, err := loadLookups(ctx, s.q)
	if err != nil {
		return err
	}
	in.ID = nil // the id is in the URL
	return validateSession(in, lk)
}

// loadSession reads one of the user's session templates with its blocks and exercises.
func loadSession(ctx context.Context, q Queries, userID, id uuid.UUID) (LibrarySession, error) {
	row, err := q.GetLibrarySession(ctx, repos.GetLibrarySessionParams{ID: id, UserID: userID})
	if err != nil {
		return LibrarySession{}, err
	}
	blocks, err := q.ListSessionBlocks(ctx, id)
	if err != nil {
		return LibrarySession{}, err
	}
	exercises, err := q.ListSessionExercises(ctx, id)
	if err != nil {
		return LibrarySession{}, err
	}
	// The rows have the same fields as a training's, so they share groupBlocks.
	tb := make([]repos.ListTrainingBlocksRow, len(blocks))
	for i, b := range blocks {
		tb[i] = repos.ListTrainingBlocksRow(b)
	}
	te := make([]repos.ListTrainingExercisesRow, len(exercises))
	for i, e := range exercises {
		te[i] = repos.ListTrainingExercisesRow(e)
	}
	return LibrarySession{GetLibrarySessionRow: row, Blocks: orEmpty(groupBlocks(tb, te)[id])}, nil
}

// sessionError is mapError for templates: a missing row is a missing session.
func sessionError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return service.NotFound("session not found", err)
	}
	return mapError(err)
}
