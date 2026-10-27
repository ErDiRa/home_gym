package app

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// CreateSession validates a new session template and saves it with its blocks.
func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, in SessionInput) (uuid.UUID, error) {
	if err := s.validateTemplate(ctx, userID, &in); err != nil {
		return uuid.Nil, err
	}
	id, err := s.library.CreateSession(ctx, userID, in)
	return id, sessionError(err)
}

// UpdateSession validates a session template and replaces its fields and blocks. Trainings it was
// added to keep their own copies.
func (s *Service) UpdateSession(ctx context.Context, userID, id uuid.UUID, in SessionInput) error {
	if err := s.validateTemplate(ctx, userID, &in); err != nil {
		return err
	}
	return sessionError(s.library.UpdateSession(ctx, userID, id, in))
}

// DeleteSession soft-deletes one of the user's session templates. Trainings it was added to keep
// their own copies.
func (s *Service) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	return sessionError(s.library.DeleteSession(ctx, userID, id))
}

// validateTemplate validates a session template against the catalog, in place.
func (s *Service) validateTemplate(ctx context.Context, userID uuid.UUID, in *SessionInput) error {
	lk, err := s.lookups(ctx)
	if err != nil {
		return err
	}
	in.ID = nil // the id is in the URL
	if err := validateSession(in, lk); err != nil {
		return err
	}
	return s.checkExercises(ctx, userID, *in)
}

// sessionError is trainingError for templates: a missing row is a missing session.
func sessionError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errs.NotFound("session not found", err)
	}
	return trainingError(err)
}
