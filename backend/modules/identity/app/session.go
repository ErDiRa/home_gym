package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

const (
	// SessionTTL is how long a freshly issued token (and cookie) stays valid.
	SessionTTL = 24 * time.Hour
	// MaxSessionAge caps how long a login can be extended, measured from the original login.
	MaxSessionAge = 7 * 24 * time.Hour
)

// Session is a signed-in user with their token. With Remember the cookie should live until
// Expiry; otherwise it is dropped when the browser closes.
type Session struct {
	Account  Account
	Token    string
	Expiry   time.Time
	Remember bool
}

// SessionExpiry returns when a token issued at now expires: SessionTTL from now, but never later
// than MaxSessionAge after the original login.
func SessionExpiry(now, authTime time.Time) time.Time {
	expiry := now.Add(SessionTTL)
	if limit := authTime.Add(MaxSessionAge); limit.Before(expiry) {
		return limit
	}
	return expiry
}

// Refresh extends the session of a token (up to MaxSessionAge after login), keeping its login time
// and kind, so the frontend can restore it without reading the HttpOnly cookie. A user deleted
// since login is NotFound, so the session can be ended.
func (s *Service) Refresh(ctx context.Context, token string) (Session, error) {
	claims, err := s.parse(token)
	if err != nil {
		return Session{}, err
	}
	acc, err := s.users.AccountByID(ctx, claims.UserID)
	if errors.Is(err, ErrUserNotFound) {
		return Session{}, errs.NotFound("account no longer exists", err)
	}
	if err != nil {
		return Session{}, err
	}
	return s.issue(acc, claims.AuthTime, claims.Remember)
}

// Authenticate checks a session token on a request: it must be valid and its user must still
// exist, so deleting a user ends their sessions at once.
func (s *Service) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	claims, err := s.parse(token)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = s.users.AccountByID(ctx, claims.UserID)
	if errors.Is(err, ErrUserNotFound) {
		return uuid.Nil, errs.Unauthorized("account no longer exists")
	}
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func (s *Service) parse(token string) (Claims, error) {
	if token == "" {
		return Claims{}, errs.Unauthorized("not authenticated")
	}
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return Claims{}, &errs.Error{Kind: errs.KindUnauthorized, Msg: "invalid or expired token", Err: err}
	}
	return claims, nil
}

// issue signs a token for the account, valid until SessionExpiry.
func (s *Service) issue(acc Account, authTime time.Time, remember bool) (Session, error) {
	expiry := SessionExpiry(s.now(), authTime)
	token, err := s.tokens.Issue(Claims{UserID: acc.ID, AuthTime: authTime, Remember: remember}, expiry)
	if err != nil {
		return Session{}, err
	}
	return Session{Account: acc, Token: token, Expiry: expiry, Remember: remember}, nil
}
