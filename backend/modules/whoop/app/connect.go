package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// Configured reports whether WHOOP can be connected at all (the app is set up).
func (s *Service) Configured() bool { return s.api.Configured() }

// AuthURL is where the browser goes to allow access; WHOOP sends it back with state.
func (s *Service) AuthURL(state string) string { return s.api.AuthURL(state) }

// Connect swaps the code WHOOP sent back for the user's tokens and stores them.
func (s *Service) Connect(ctx context.Context, userID uuid.UUID, code string) error {
	tokens, err := s.api.ExchangeCode(ctx, code)
	if err != nil {
		return apiError(err)
	}
	return s.connections.SaveConnection(ctx, userID, tokens)
}

// Connected reports whether the user has connected WHOOP.
func (s *Service) Connected(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, err := s.connections.Connection(ctx, userID)
	if errors.Is(err, ErrNotConnected) {
		return false, nil
	}
	return err == nil, err
}

// Disconnect forgets the user's tokens; the data of linked workouts stays.
func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	return s.connections.DeleteConnection(ctx, userID)
}

// accessToken returns a valid access token of the user, refreshing it when it is about to expire.
// The refresh token rotates, so the new one is stored right away.
func (s *Service) accessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	tokens, err := s.connections.Connection(ctx, userID)
	if errors.Is(err, ErrNotConnected) {
		return "", errs.Invalid("WHOOP is not connected")
	}
	if err != nil {
		return "", err
	}
	if tokens.ExpiresAt.Sub(s.now()) > time.Minute {
		return tokens.AccessToken, nil
	}
	if tokens, err = s.api.Refresh(ctx, tokens.RefreshToken); err != nil {
		return "", apiError(err)
	}
	if err := s.connections.SaveConnection(ctx, userID, tokens); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// apiError turns the API's errors into the ones the caller sees: WHOOP rejecting the credentials
// asks the user to connect again.
func apiError(err error) error {
	switch {
	case errors.Is(err, ErrRejected):
		return &errs.Error{Kind: errs.KindUnauthorized, Msg: "WHOOP rejected the connection, connect it again", Err: err}
	case errors.Is(err, ErrUnknownActivity):
		return errs.InvalidCause("unknown WHOOP activity", err)
	default:
		return err
	}
}
