package cycle

import (
	"strings"
	"testing"

	"github.com/erdira/homegym/backend/internal/service"
)

// validCycle is a valid 8-week cycle with two goals.
func validCycle() Input {
	return Input{Name: "Mobility", StartDate: "2026-03-02", Weeks: 8, Goals: []GoalInput{
		{Title: "Better hips", Target: "Deep squat hold 2 min"},
		{Title: "Better elbow extension", Status: GoalAchieved},
	}}
}

func TestValidateCycleTrims(t *testing.T) {
	in := validCycle()
	in.Name, in.Description = "  Mobility ", " Hips and elbows\n"
	in.Goals[0].Title, in.Goals[0].Target, in.Goals[0].Notes = " Better hips ", " 2 min ", " daily "

	if err := Validate(&in); err != nil {
		t.Fatal(err)
	}
	g := in.Goals[0]
	if in.Name != "Mobility" || in.Description != "Hips and elbows" || g.Title != "Better hips" || g.Target != "2 min" || g.Notes != "daily" {
		t.Errorf("not trimmed: %+v", in)
	}
	// An empty status is open; a set one is kept.
	if g.Status != GoalOpen || in.Goals[1].Status != GoalAchieved {
		t.Errorf("statuses = %q, %q", g.Status, in.Goals[1].Status)
	}
	if !in.Start.Valid || in.Start.Time.Day() != 2 {
		t.Errorf("start = %+v", in.Start)
	}
}

func TestValidateCycleRejects(t *testing.T) {
	tooMany := make([]GoalInput, maxGoals+1)
	for i := range tooMany {
		tooMany[i] = GoalInput{Title: "Goal"}
	}
	cases := []struct {
		name   string
		change func(*Input)
		msg    string
	}{
		{"no name", func(c *Input) { c.Name = "  " }, "name is required (max 100 characters)"},
		{"long name", func(c *Input) { c.Name = strings.Repeat("x", 101) }, "name is required (max 100 characters)"},
		{"long description", func(c *Input) { c.Description = strings.Repeat("x", 2001) }, "description is too long (max 2000 characters)"},
		{"no start", func(c *Input) { c.StartDate = "" }, "start_date is required (YYYY-MM-DD)"},
		{"not a Monday", func(c *Input) { c.StartDate = "2026-03-03" }, "start_date must be a Monday"},
		{"0 weeks", func(c *Input) { c.Weeks = 0 }, "weeks must be 1-52"},
		{"53 weeks", func(c *Input) { c.Weeks = 53 }, "weeks must be 1-52"},
		{"too many goals", func(c *Input) { c.Goals = tooMany }, "a cycle has at most 20 goals"},
		{"empty goal title", func(c *Input) { c.Goals[1].Title = " " }, "goal 2: a title is required (max 200 characters)"},
		{"long target", func(c *Input) { c.Goals[0].Target = strings.Repeat("x", 201) }, "goal 1: the target is too long (max 200 characters)"},
		{"long notes", func(c *Input) { c.Goals[0].Notes = strings.Repeat("x", 2001) }, "goal 1: the notes are too long (max 2000 characters)"},
		{"unknown status", func(c *Input) { c.Goals[1].Status = "done" }, "goal 2: the status must be open, achieved or missed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validCycle()
			tc.change(&in)
			wantError(t, Validate(&in), service.KindInvalid, tc.msg)
		})
	}
}
