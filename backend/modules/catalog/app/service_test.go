package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// fakeRepo implements the three repositories in memory; every call fails with err when it is set.
type fakeRepo struct {
	err       error
	created   []ExerciseInput
	updated   []uuid.UUID // the ids updated
	ratings   []RatingInput
	repUnits  []UnitInput
	loadUnits []UnitInput
}

func (f *fakeRepo) CreateExercise(_ context.Context, _ uuid.UUID, in ExerciseInput) (Exercise, error) {
	if f.err != nil {
		return Exercise{}, f.err
	}
	f.created = append(f.created, in)
	return Exercise{ID: uuid.New(), Name: in.Name, Type: in.Type, VideoURLs: in.VideoURLs}, nil
}

func (f *fakeRepo) UpdateExercise(_ context.Context, _, id uuid.UUID, in ExerciseInput) (Exercise, error) {
	if f.err != nil {
		return Exercise{}, f.err
	}
	f.updated = append(f.updated, id)
	return Exercise{ID: id, Name: in.Name}, nil
}

func (f *fakeRepo) RateExercise(_ context.Context, userID, exerciseID uuid.UUID, in RatingInput) (Rating, error) {
	if f.err != nil {
		return Rating{}, f.err
	}
	f.ratings = append(f.ratings, in)
	return Rating{ID: uuid.New(), UserID: userID, ExerciseID: exerciseID, Rating: in.Rating, Notes: in.Notes}, nil
}

func (f *fakeRepo) CreateFocus(_ context.Context, in FocusInput) (Focus, error) {
	if f.err != nil {
		return Focus{}, f.err
	}
	return Focus{ID: uuid.New(), Name: in.Name, Alias: in.Alias}, nil
}

func (f *fakeRepo) CreateRepUnit(_ context.Context, in UnitInput) (Unit, error) {
	if f.err != nil {
		return Unit{}, f.err
	}
	f.repUnits = append(f.repUnits, in)
	return Unit{ID: uuid.New(), Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue}, nil
}

func (f *fakeRepo) CreateLoadUnit(_ context.Context, in UnitInput) (Unit, error) {
	if f.err != nil {
		return Unit{}, f.err
	}
	f.loadUnits = append(f.loadUnits, in)
	return Unit{ID: uuid.New(), Name: in.Name, MinValue: in.MinValue, MaxValue: in.MaxValue}, nil
}

func newService(f *fakeRepo) *Service { return NewService(f, f, f) }

// wantError fails the test unless err is a *errs.Error of kind with message msg.
func wantError(t *testing.T, err error, kind errs.Kind, msg string) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v, want an errs.Error %q", err, msg)
	}
	if e.Kind != kind || e.Msg != msg {
		t.Fatalf("err = {kind %d, %q}, want {kind %d, %q}", e.Kind, e.Msg, kind, msg)
	}
}

func ptr[T any](v T) *T { return &v }

func validExercise() ExerciseInput {
	return ExerciseInput{Name: "Squat", JointIDs: []uuid.UUID{uuid.New()}, MuscleGroupIDs: []uuid.UUID{uuid.New()}, Type: "strength"}
}

func TestCreateExercise(t *testing.T) {
	f := &fakeRepo{}
	knee, focus := uuid.New(), uuid.New()
	in := validExercise()
	in.JointIDs = []uuid.UUID{knee, knee}
	in.FocusIDs = []uuid.UUID{focus, focus}
	in.VideoURLs = []string{" https://example.com/a ", "", "  ", "http://example.com/b"}

	got, err := newService(f).CreateExercise(context.Background(), uuid.New(), in)
	if err != nil {
		t.Fatal(err)
	}
	saved := f.created[0]
	if !slices.Equal(saved.JointIDs, []uuid.UUID{knee}) || !slices.Equal(saved.FocusIDs, []uuid.UUID{focus}) {
		t.Errorf("ids not deduplicated: %v, %v", saved.JointIDs, saved.FocusIDs)
	}
	if want := []string{"https://example.com/a", "http://example.com/b"}; !slices.Equal(got.VideoURLs, want) {
		t.Errorf("videos = %q, want %q", got.VideoURLs, want)
	}
}

func TestCreateExerciseWithoutLinks(t *testing.T) {
	f := &fakeRepo{}
	if _, err := newService(f).CreateExercise(context.Background(), uuid.New(), ExerciseInput{Name: "General Warmup 1", Type: "mobility"}); err != nil {
		t.Fatal(err)
	}
	// Empty lists, not nil: they become the queries' array parameters.
	if saved := f.created[0]; saved.JointIDs == nil || len(saved.JointIDs) != 0 || len(saved.MuscleGroupIDs) != 0 {
		t.Errorf("ids = %v, %v, want empty lists", saved.JointIDs, saved.MuscleGroupIDs)
	}
}

func TestCreateExerciseRejects(t *testing.T) {
	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = "https://example.com/v"
	}
	tests := []struct {
		name    string
		edit    func(*ExerciseInput)
		wantMsg string
	}{
		{"no name", func(in *ExerciseInput) { in.Name = "" }, "name is required"},
		{"bad type", func(in *ExerciseInput) { in.Type = "cardio" }, "type must be mobility or strength"},
		{"not http", func(in *ExerciseInput) { in.VideoURLs = []string{"ftp://example.com/v"} }, "video links must be http(s) URLs (max 500 characters)"},
		{"no host", func(in *ExerciseInput) { in.VideoURLs = []string{"https://"} }, "video links must be http(s) URLs (max 500 characters)"},
		{"long url", func(in *ExerciseInput) { in.VideoURLs = []string{"https://example.com/" + strings.Repeat("v", 500)} }, "video links must be http(s) URLs (max 500 characters)"},
		{"too many videos", func(in *ExerciseInput) { in.VideoURLs = tooMany }, "an exercise can have at most 10 video links"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRepo{}
			in := validExercise()
			tt.edit(&in)
			_, err := newService(f).CreateExercise(context.Background(), uuid.New(), in)
			wantError(t, err, errs.KindInvalid, tt.wantMsg)
			if len(f.created) != 0 {
				t.Error("created an exercise anyway")
			}
		})
	}

	_, err := newService(&fakeRepo{err: ErrUnknownReference}).CreateExercise(context.Background(), uuid.New(), validExercise())
	wantError(t, err, errs.KindInvalid, "unknown joint, muscle group or focus")
}

func TestUpdateExercise(t *testing.T) {
	f := &fakeRepo{}
	id := uuid.New()
	if _, err := newService(f).UpdateExercise(context.Background(), uuid.New(), id, validExercise()); err != nil {
		t.Fatal(err)
	}
	if f.updated[0] != id {
		t.Errorf("updated %v, want %v", f.updated[0], id)
	}

	s := newService(&fakeRepo{err: ErrNotFound})
	_, err := s.UpdateExercise(context.Background(), uuid.New(), id, validExercise())
	wantError(t, err, errs.KindNotFound, "exercise not found")
	_, err = s.UpdateExercise(context.Background(), uuid.New(), id, ExerciseInput{})
	wantError(t, err, errs.KindInvalid, "name is required")
}

func TestRateExercise(t *testing.T) {
	f := &fakeRepo{}
	user, exercise := uuid.New(), uuid.New()
	got, err := newService(f).RateExercise(context.Background(), user, exercise, RatingInput{Rating: 4, Notes: " knees ok "})
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != user || got.ExerciseID != exercise || got.Rating != 4 || got.Notes != "knees ok" {
		t.Errorf("rating = %+v", got)
	}

	for _, rating := range []int32{0, 6, -1} {
		f := &fakeRepo{}
		_, err := newService(f).RateExercise(context.Background(), uuid.New(), uuid.New(), RatingInput{Rating: rating})
		wantError(t, err, errs.KindInvalid, "rating must be between 1 and 5")
		if len(f.ratings) != 0 {
			t.Errorf("rating %d was stored", rating)
		}
	}

	// Another user's or an unknown exercise.
	_, err = newService(&fakeRepo{err: ErrNotFound}).RateExercise(context.Background(), uuid.New(), uuid.New(), RatingInput{Rating: 3})
	wantError(t, err, errs.KindNotFound, "exercise not found")
}

func TestCreateFocus(t *testing.T) {
	got, err := newService(&fakeRepo{}).CreateFocus(context.Background(), FocusInput{Name: " Hüftmobilität ", Alias: " HÜFTE "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Hüftmobilität" || got.Alias != "HÜFTE" {
		t.Errorf("focus = %+v, want trimmed name and alias", got)
	}

	tests := []struct {
		name     string
		in       FocusInput
		err      error
		wantKind errs.Kind
		wantMsg  string
	}{
		{"no name", FocusInput{Alias: "HIP"}, nil, errs.KindInvalid, "name is required (max 60 characters)"},
		{"long name", FocusInput{Name: strings.Repeat("x", 61), Alias: "HIP"}, nil, errs.KindInvalid, "name is required (max 60 characters)"},
		{"no alias", FocusInput{Name: "Hip"}, nil, errs.KindInvalid, "alias is required (max 10 characters)"},
		{"long alias", FocusInput{Name: "Hip", Alias: strings.Repeat("x", 11)}, nil, errs.KindInvalid, "alias is required (max 10 characters)"},
		{"duplicate", FocusInput{Name: "Hip", Alias: "HIP"}, ErrDuplicate, errs.KindConflict, "this focus already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newService(&fakeRepo{err: tt.err}).CreateFocus(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
		})
	}
}

func TestCreateUnit(t *testing.T) {
	f := &fakeRepo{}
	s := newService(f)
	rpe, err := s.CreateLoadUnit(context.Background(), UnitInput{Name: " RPE ", MinValue: ptr(1.0), MaxValue: ptr(10.0)})
	if err != nil {
		t.Fatal(err)
	}
	if rpe.Name != "RPE" || *rpe.MaxValue != 10 {
		t.Errorf("load unit = %+v", rpe)
	}
	if _, err := s.CreateRepUnit(context.Background(), UnitInput{Name: "s"}); err != nil {
		t.Fatal(err)
	}
	if len(f.repUnits) != 1 || f.repUnits[0].MinValue != nil {
		t.Errorf("rep units = %+v, want one unbounded", f.repUnits)
	}

	tests := []struct {
		name     string
		in       UnitInput
		err      error
		wantKind errs.Kind
		wantMsg  string
	}{
		{"blank name", UnitInput{Name: "  "}, nil, errs.KindInvalid, "name is required (max 30 characters)"},
		{"long name", UnitInput{Name: strings.Repeat("x", 31)}, nil, errs.KindInvalid, "name is required (max 30 characters)"},
		{"negative bound", UnitInput{Name: "kg", MinValue: ptr(-1.0)}, nil, errs.KindInvalid, "bounds must be between 0 and 99999"},
		{"bound too high", UnitInput{Name: "kg", MaxValue: ptr(100000.0)}, nil, errs.KindInvalid, "bounds must be between 0 and 99999"},
		{"max below min", UnitInput{Name: "kg", MinValue: ptr(5.0), MaxValue: ptr(4.0)}, nil, errs.KindInvalid, "max_value must not be lower than min_value"},
		{"duplicate", UnitInput{Name: "kg"}, ErrDuplicate, errs.KindConflict, "this unit already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newService(&fakeRepo{err: tt.err})
			_, err := s.CreateRepUnit(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
			_, err = s.CreateLoadUnit(context.Background(), tt.in)
			wantError(t, err, tt.wantKind, tt.wantMsg)
		})
	}
}
