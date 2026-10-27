package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/erdira/homegym/backend/common/errs"
)

// fakeUsers implements UserRepository over a map by email.
type fakeUsers struct {
	users map[string]User
	err   error // returned by every call when set
}

func (f *fakeUsers) UserByEmail(_ context.Context, email string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	u, ok := f.users[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeUsers) AccountByID(_ context.Context, id uuid.UUID) (Account, error) {
	if f.err != nil {
		return Account{}, f.err
	}
	for _, u := range f.users {
		if u.ID == id {
			return u.Account, nil
		}
	}
	return Account{}, ErrUserNotFound
}

// fakeTokens "signs" claims by keeping them under a random token.
type fakeTokens struct {
	issued  map[string]Claims
	expires map[string]time.Time
}

func (f *fakeTokens) Issue(c Claims, expiry time.Time) (string, error) {
	token := uuid.NewString()
	f.issued[token], f.expires[token] = c, expiry
	return token, nil
}

func (f *fakeTokens) Parse(token string) (Claims, error) {
	c, ok := f.issued[token]
	if !ok {
		return Claims{}, errors.New("unknown token")
	}
	return c, nil
}

var now = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

func newService(t *testing.T) (*Service, *fakeUsers, *fakeTokens, Account) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("my-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	acc := Account{ID: uuid.New(), Email: "me@example.com"}
	users := &fakeUsers{users: map[string]User{acc.Email: {Account: acc, PasswordHash: string(hash)}}}
	tokens := &fakeTokens{issued: map[string]Claims{}, expires: map[string]time.Time{}}
	s := NewService(users, tokens)
	s.now = func() time.Time { return now }
	return s, users, tokens, acc
}

func wantError(t *testing.T, err error, kind errs.Kind, msg string) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Kind != kind || e.Msg != msg {
		t.Errorf("err = %v, want %v %q", err, kind, msg)
	}
}

func TestLogin(t *testing.T) {
	s, _, tokens, acc := newService(t)

	session, err := s.Login(context.Background(), " Me@Example.com ", "my-pass", true)
	if err != nil {
		t.Fatal(err)
	}
	if session.Account != acc || !session.Remember || !session.Expiry.Equal(now.Add(SessionTTL)) {
		t.Errorf("session = %+v", session)
	}
	if c := tokens.issued[session.Token]; c.UserID != acc.ID || !c.AuthTime.Equal(now) || !c.Remember {
		t.Errorf("claims = %+v", c)
	}
}

func TestLoginRejects(t *testing.T) {
	s, users, _, _ := newService(t)
	for _, c := range []struct{ name, email, password string }{
		{"wrong password", "me@example.com", "wrong"},
		{"unknown email", "nobody@example.com", "my-pass"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.Login(context.Background(), c.email, c.password, false)
			wantError(t, err, errs.KindUnauthorized, "invalid email or password")
		})
	}

	// A failing database is not a wrong password.
	users.err = errors.New("connection refused")
	_, err := s.Login(context.Background(), "me@example.com", "my-pass", false)
	var e *errs.Error
	if err == nil || errors.As(err, &e) {
		t.Errorf("err = %v, want the database error", err)
	}
}

func TestRefreshKeepsLoginTime(t *testing.T) {
	s, users, tokens, acc := newService(t)
	login, _ := s.Login(context.Background(), acc.Email, "my-pass", false)

	// Six and a half days later the session only runs to the 7-day cap.
	s.now = func() time.Time { return now.Add(MaxSessionAge - 12*time.Hour) }
	session, err := s.Refresh(context.Background(), login.Token)
	if err != nil {
		t.Fatal(err)
	}
	if c := tokens.issued[session.Token]; !c.AuthTime.Equal(now) || c.Remember || !session.Expiry.Equal(now.Add(MaxSessionAge)) {
		t.Errorf("refreshed = %+v, claims %+v", session, c)
	}

	_, err = s.Refresh(context.Background(), "nope")
	wantError(t, err, errs.KindUnauthorized, "invalid or expired token")
	_, err = s.Refresh(context.Background(), "")
	wantError(t, err, errs.KindUnauthorized, "not authenticated")

	delete(users.users, acc.Email)
	_, err = s.Refresh(context.Background(), login.Token)
	wantError(t, err, errs.KindNotFound, "account no longer exists")
}

func TestAuthenticate(t *testing.T) {
	s, users, _, acc := newService(t)
	login, _ := s.Login(context.Background(), acc.Email, "my-pass", false)

	if id, err := s.Authenticate(context.Background(), login.Token); err != nil || id != acc.ID {
		t.Errorf("Authenticate = %v, %v", id, err)
	}
	_, err := s.Authenticate(context.Background(), "nope")
	wantError(t, err, errs.KindUnauthorized, "invalid or expired token")

	delete(users.users, acc.Email)
	_, err = s.Authenticate(context.Background(), login.Token)
	wantError(t, err, errs.KindUnauthorized, "account no longer exists")
}

func TestSessionExpiry(t *testing.T) {
	login := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Shortly after login the session is extended by the full TTL.
	at := login.Add(time.Hour)
	if got, want := SessionExpiry(at, login), at.Add(SessionTTL); !got.Equal(want) {
		t.Errorf("early refresh = %v, want %v", got, want)
	}
	// Near the cap the extension is cut off at login + MaxSessionAge.
	at = login.Add(MaxSessionAge - time.Hour)
	if got, want := SessionExpiry(at, login), login.Add(MaxSessionAge); !got.Equal(want) {
		t.Errorf("capped refresh = %v, want %v", got, want)
	}
}
