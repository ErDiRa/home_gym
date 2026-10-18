package exercise

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/erdira/homegym/backend/internal/service"
)

func validInput() Input {
	return Input{Name: "Squat", JointIDs: []uuid.UUID{uuid.New()}, MuscleGroupIDs: []uuid.UUID{uuid.New()}, Type: "strength"}
}

func TestCreate(t *testing.T) {
	f := &fakeQueries{}
	knee, focus := uuid.New(), uuid.New()
	in := validInput()
	in.JointIDs = []uuid.UUID{knee, knee}
	in.FocusIDs = []uuid.UUID{focus, focus}
	in.VideoURLs = []string{" https://example.com/a ", "", "  ", "http://example.com/b"}

	userID := uuid.New()
	got, err := (&Service{q: f}).Create(context.Background(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	params := f.created[0]
	if params.UserID != userID {
		t.Errorf("created for %v, want %v", params.UserID, userID)
	}
	if !slices.Equal(params.JointIds, []uuid.UUID{knee}) || !slices.Equal(params.FocusIds, []uuid.UUID{focus}) {
		t.Errorf("ids not deduplicated: %v, %v", params.JointIds, params.FocusIds)
	}
	if want := []string{"https://example.com/a", "http://example.com/b"}; !slices.Equal(got.VideoUrls, want) {
		t.Errorf("videos = %q, want %q", got.VideoUrls, want)
	}
}

func TestCreateRejects(t *testing.T) {
	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = "https://example.com/v"
	}
	tests := []struct {
		name    string
		edit    func(*Input)
		wantMsg string
	}{
		{"no name", func(in *Input) { in.Name = "" }, "name, joint_ids and muscle_group_ids are required"},
		{"no joints", func(in *Input) { in.JointIDs = nil }, "name, joint_ids and muscle_group_ids are required"},
		{"no muscle groups", func(in *Input) { in.MuscleGroupIDs = []uuid.UUID{} }, "name, joint_ids and muscle_group_ids are required"},
		{"bad type", func(in *Input) { in.Type = "cardio" }, "type must be mobility or strength"},
		{"not http", func(in *Input) { in.VideoURLs = []string{"ftp://example.com/v"} }, "video links must be http(s) URLs (max 500 characters)"},
		{"no host", func(in *Input) { in.VideoURLs = []string{"https://"} }, "video links must be http(s) URLs (max 500 characters)"},
		{"long url", func(in *Input) { in.VideoURLs = []string{"https://example.com/" + strings.Repeat("v", 500)} }, "video links must be http(s) URLs (max 500 characters)"},
		{"too many videos", func(in *Input) { in.VideoURLs = tooMany }, "an exercise can have at most 10 video links"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeQueries{}
			in := validInput()
			tt.edit(&in)
			_, err := (&Service{q: f}).Create(context.Background(), uuid.New(), in)
			wantError(t, err, service.KindInvalid, tt.wantMsg)
			if len(f.created) != 0 {
				t.Error("created an exercise anyway")
			}
		})
	}
}

func TestCreateUnknownLink(t *testing.T) {
	f := &fakeQueries{err: &pgconn.PgError{Code: "23503"}}
	_, err := (&Service{q: f}).Create(context.Background(), uuid.New(), validInput())
	wantError(t, err, service.KindInvalid, "unknown joint, muscle group or focus")
}
