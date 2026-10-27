package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// fakeAPI answers with the activity and recovery set, refreshing to "new" tokens.
type fakeAPI struct {
	activity     Activity
	recovery     *Recovery
	refreshed    []string // the refresh tokens used
	reject       bool     // WHOOP rejects the refresh token
	recoveryFrom time.Time
}

func (f *fakeAPI) Configured() bool      { return true }
func (f *fakeAPI) AuthURL(string) string { return "https://whoop/auth" }
func (f *fakeAPI) ExchangeCode(context.Context, string) (Tokens, error) {
	return Tokens{AccessToken: "first", RefreshToken: "r1", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeAPI) Refresh(_ context.Context, refreshToken string) (Tokens, error) {
	if f.reject {
		return Tokens{}, ErrRejected
	}
	f.refreshed = append(f.refreshed, refreshToken)
	return Tokens{AccessToken: "new", RefreshToken: "rotated", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (f *fakeAPI) Activities(context.Context, string, time.Time, time.Time) ([]Activity, error) {
	return []Activity{f.activity}, nil
}

func (f *fakeAPI) Activity(_ context.Context, _ string, id uuid.UUID) (Activity, error) {
	if id != f.activity.ID {
		return Activity{}, ErrUnknownActivity
	}
	return f.activity, nil
}

func (f *fakeAPI) LatestRecovery(_ context.Context, _ string, from, _ time.Time) (*Recovery, error) {
	f.recoveryFrom = from
	return f.recovery, nil
}

// fakeRepos stores one user's connection and the saved data; workout is the user's workout.
type fakeRepos struct {
	tokens  *Tokens
	saved   []Data
	workout uuid.UUID
}

func (f *fakeRepos) Connection(context.Context, uuid.UUID) (Tokens, error) {
	if f.tokens == nil {
		return Tokens{}, ErrNotConnected
	}
	return *f.tokens, nil
}

func (f *fakeRepos) SaveConnection(_ context.Context, _ uuid.UUID, t Tokens) error {
	f.tokens = &t
	return nil
}

func (f *fakeRepos) DeleteConnection(context.Context, uuid.UUID) error {
	f.tokens = nil
	return nil
}

func (f *fakeRepos) SaveData(_ context.Context, d Data) (Data, error) {
	f.saved = append(f.saved, d)
	return d, nil
}

func (f *fakeRepos) Data(context.Context, uuid.UUID) (*Data, error) {
	if len(f.saved) == 0 {
		return nil, nil
	}
	return &f.saved[len(f.saved)-1], nil
}

func (f *fakeRepos) DeleteData(context.Context, uuid.UUID) (bool, error) {
	return len(f.saved) > 0, nil
}

func (f *fakeRepos) UserOwnsWorkout(_ context.Context, _, workoutID uuid.UUID) (bool, error) {
	return workoutID == f.workout, nil
}

func newTestService(tokens *Tokens) (*Service, *fakeAPI, *fakeRepos) {
	api, repos := &fakeAPI{}, &fakeRepos{tokens: tokens, workout: uuid.New()}
	return NewService(api, repos, repos, repos), api, repos
}

func wantKind(t *testing.T, err error, kind errs.Kind) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Kind != kind {
		t.Errorf("err = %v, want an errs.Error of kind %d", err, kind)
	}
}

func valid() *Tokens {
	return &Tokens{AccessToken: "valid", RefreshToken: "r1", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestLink(t *testing.T) {
	s, api, repos := newTestService(valid())
	start := time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	api.activity = Activity{ID: uuid.New(), SportName: "weightlifting", Start: start, ScoreState: "SCORED"}
	api.recovery = &Recovery{Score: 68}

	d, err := s.Link(context.Background(), uuid.New(), repos.workout, api.activity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.WorkoutID != repos.workout || d.SportName != "weightlifting" || d.Recovery.Score != 68 {
		t.Errorf("linked = %+v", d)
	}
	// The recovery is the newest of the day before the activity.
	if !api.recoveryFrom.Equal(start.Add(-24 * time.Hour)) {
		t.Errorf("recovery from %v", api.recoveryFrom)
	}

	// Another user's workout looks missing; an activity WHOOP doesn't know is invalid.
	_, err = s.Link(context.Background(), uuid.New(), uuid.New(), api.activity.ID)
	wantKind(t, err, errs.KindNotFound)
	_, err = s.Link(context.Background(), uuid.New(), repos.workout, uuid.New())
	wantKind(t, err, errs.KindInvalid)
}

func TestAccessTokenRefresh(t *testing.T) {
	s, api, repos := newTestService(&Tokens{AccessToken: "old", RefreshToken: "r1", ExpiresAt: time.Now()})
	token, err := s.accessToken(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	// The rotated refresh token is stored right away.
	if token != "new" || api.refreshed[0] != "r1" || repos.tokens.RefreshToken != "rotated" {
		t.Errorf("token %q, stored %+v", token, repos.tokens)
	}

	// A refresh token WHOOP no longer accepts asks to reconnect.
	repos.tokens.ExpiresAt = time.Now()
	api.reject = true
	_, err = s.accessToken(context.Background(), uuid.New())
	wantKind(t, err, errs.KindUnauthorized)

	// Without a connection there is nothing to call WHOOP with.
	repos.tokens = nil
	_, err = s.accessToken(context.Background(), uuid.New())
	wantKind(t, err, errs.KindInvalid)
}

func TestDataIsTheUsers(t *testing.T) {
	s, _, repos := newTestService(valid())
	repos.saved = []Data{{WorkoutID: repos.workout}}
	if d, err := s.Data(context.Background(), uuid.New(), uuid.New()); d != nil || err != nil {
		t.Errorf("another user's workout's data = %+v, %v; want none", d, err)
	}
	wantKind(t, s.Unlink(context.Background(), uuid.New(), uuid.New()), errs.KindNotFound)
	if err := s.Unlink(context.Background(), uuid.New(), repos.workout); err != nil {
		t.Errorf("unlink = %v", err)
	}
	_, err := s.Candidates(context.Background(), uuid.New(), repos.workout, time.Now(), time.Now().Add(72*time.Hour))
	wantKind(t, err, errs.KindInvalid)
}
