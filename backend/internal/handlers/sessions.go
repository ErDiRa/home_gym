package handlers

import (
	"context"

	"github.com/erdira/homegym/backend/internal/api"
	"github.com/erdira/homegym/backend/internal/service/training"
	trainingrepos "github.com/erdira/homegym/backend/internal/service/training/repos"
)

// ListSessions returns the user's session templates with their size, for the library.
func (h *Handler) ListSessions(ctx context.Context, _ api.ListSessionsRequestObject) (api.ListSessionsResponseObject, error) {
	rows, err := listOwn(ctx, h.trainings.ListSessions, func(s trainingrepos.ListLibrarySessionsRow) api.LibrarySession {
		return api.LibrarySession{
			Id: s.ID, Name: s.Name, Notes: s.Notes, UpdatedAt: timestamp(s.UpdatedAt),
			BlockCount: s.BlockCount, ExerciseCount: s.ExerciseCount,
		}
	})
	return api.ListSessions200JSONResponse(rows), err
}

// GetSession returns one of the user's session templates with its blocks and exercises.
func (h *Handler) GetSession(ctx context.Context, req api.GetSessionRequestObject) (api.GetSessionResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	s, err := h.trainings.GetSession(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	return api.GetSession200JSONResponse(librarySessionToAPI(s)), nil
}

// CreateSession saves a new session template with its blocks.
func (h *Handler) CreateSession(ctx context.Context, req api.CreateSessionRequestObject) (api.CreateSessionResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	s, err := h.trainings.CreateSession(ctx, userID, sessionInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateSession201JSONResponse(librarySessionToAPI(s)), nil
}

// UpdateSession replaces a session template's fields and blocks; trainings keep their copies.
func (h *Handler) UpdateSession(ctx context.Context, req api.UpdateSessionRequestObject) (api.UpdateSessionResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	s, err := h.trainings.UpdateSession(ctx, userID, req.Id, sessionInputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateSession200JSONResponse(librarySessionToAPI(s)), nil
}

// DeleteSession soft-deletes a session template; trainings keep their copies.
func (h *Handler) DeleteSession(ctx context.Context, req api.DeleteSessionRequestObject) (api.DeleteSessionResponseObject, error) {
	userID, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.trainings.DeleteSession(ctx, userID, req.Id); err != nil {
		return nil, err
	}
	return api.DeleteSession204Response{}, nil
}

func librarySessionToAPI(s training.LibrarySession) api.LibrarySessionDetail {
	return api.LibrarySessionDetail{
		Id: s.ID, Name: s.Name, Notes: s.Notes, CreatedAt: timestamp(s.CreatedAt), UpdatedAt: timestamp(s.UpdatedAt),
		Blocks: mapSlice(s.Blocks, blockToAPI),
	}
}
