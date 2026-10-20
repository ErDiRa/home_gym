package workout

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/workout/repos"
)

func TestLogResult(t *testing.T) {
	tests := []struct {
		name string
		in   ResultInput
		want repos.UpsertWorkoutResultParams // WorkoutID and BlockPosition are filled in
	}{
		{"for time", ResultInput{ScoreType: scoreForTime, TimeSeconds: ptr[int32](522), Rounds: ptr[int32](3), Rx: true},
			repos.UpsertWorkoutResultParams{ScoreType: scoreForTime, TimeSeconds: ptr[int32](522), Rx: true}},
		{"for time capped", ResultInput{ScoreType: scoreForTime, Reps: ptr[int32](143)},
			repos.UpsertWorkoutResultParams{ScoreType: scoreForTime, Reps: ptr[int32](143)}},
		{"amrap", ResultInput{ScoreType: scoreAMRAP, Rounds: ptr[int32](5), Reps: ptr[int32](12), TimeSeconds: ptr[int32](720), Notes: "  scaled pull-ups "},
			repos.UpsertWorkoutResultParams{ScoreType: scoreAMRAP, Rounds: ptr[int32](5), Reps: ptr[int32](12), Notes: "scaled pull-ups"}},
		{"amrap of whole rounds", ResultInput{ScoreType: scoreAMRAP, Rounds: ptr[int32](7)},
			repos.UpsertWorkoutResultParams{ScoreType: scoreAMRAP, Rounds: ptr[int32](7)}},
		{"max reps", ResultInput{ScoreType: scoreMaxReps, Reps: ptr[int32](150), Calories: ptr[int32](3)},
			repos.UpsertWorkoutResultParams{ScoreType: scoreMaxReps, Reps: ptr[int32](150)}},
		{"max calories", ResultInput{ScoreType: scoreMaxCalories, Calories: ptr[int32](42), Reps: ptr[int32](1)},
			repos.UpsertWorkoutResultParams{ScoreType: scoreMaxCalories, Calories: ptr[int32](42)}},
		{"emom", ResultInput{ScoreType: scoreEMOM, Rounds: ptr[int32](10), Reps: ptr[int32](4)},
			repos.UpsertWorkoutResultParams{ScoreType: scoreEMOM, Rounds: ptr[int32](10)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			f.workout = &repos.Workout{ID: uuid.New()}
			tt.in.BlockPosition = 2
			if _, err := (&Service{q: f}).LogResult(context.Background(), uuid.New(), f.workout.ID, tt.in); err != nil {
				t.Fatal(err)
			}
			tt.want.WorkoutID, tt.want.BlockPosition = f.workout.ID, 2
			if got := f.resultUpserts[0]; !equalResult(got, tt.want) {
				t.Errorf("upserted %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLogResultRejects(t *testing.T) {
	tests := []struct {
		name, msg string
		in        ResultInput
	}{
		{"no block", "block_position must be 1 or more", ResultInput{ScoreType: scoreMaxReps, Reps: ptr[int32](5)}},
		{"unknown format", "unknown score type", ResultInput{BlockPosition: 1, ScoreType: "tabata", Reps: ptr[int32](5)}},
		{"for time without anything", "enter the time, or the reps if the time cap was hit", ResultInput{BlockPosition: 1, ScoreType: scoreForTime}},
		{"for time with time and reps", "enter the time, or the reps if the time cap was hit, not both", ResultInput{BlockPosition: 1, ScoreType: scoreForTime, TimeSeconds: ptr[int32](60), Reps: ptr[int32](5)}},
		{"for time of zero seconds", "the time must be between 1 and 86400", ResultInput{BlockPosition: 1, ScoreType: scoreForTime, TimeSeconds: ptr[int32](0)}},
		{"amrap without rounds", "enter the rounds", ResultInput{BlockPosition: 1, ScoreType: scoreAMRAP, Reps: ptr[int32](12)}},
		{"negative reps", "the reps must be between 0 and 100000", ResultInput{BlockPosition: 1, ScoreType: scoreMaxReps, Reps: ptr[int32](-1)}},
		{"max calories without calories", "enter the calories", ResultInput{BlockPosition: 1, ScoreType: scoreMaxCalories, Reps: ptr[int32](12)}},
		{"emom without rounds", "enter the rounds completed", ResultInput{BlockPosition: 1, ScoreType: scoreEMOM}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, _ := newFake()
			f.workout = &repos.Workout{ID: uuid.New()}
			_, err := (&Service{q: f}).LogResult(context.Background(), uuid.New(), f.workout.ID, tt.in)
			wantError(t, err, service.KindInvalid, tt.msg)
			if len(f.resultUpserts) != 0 {
				t.Error("upserted an invalid result")
			}
		})
	}
}

func TestLogResultOtherUsersWorkout(t *testing.T) {
	f, _, _ := newFake() // no workout: GetWorkout finds none
	_, err := (&Service{q: f}).LogResult(context.Background(), uuid.New(), uuid.New(), ResultInput{BlockPosition: 1, ScoreType: scoreMaxReps, Reps: ptr[int32](5)})
	wantError(t, err, service.KindNotFound, "workout not found")
}

func TestDeleteResult(t *testing.T) {
	f, _, _ := newFake()
	f.workout = &repos.Workout{ID: uuid.New()}
	err := (&Service{q: f}).DeleteResult(context.Background(), uuid.New(), f.workout.ID, uuid.New())
	wantError(t, err, service.KindNotFound, "result not found")

	f.deleteRows = 1
	if err := (&Service{q: f}).DeleteResult(context.Background(), uuid.New(), f.workout.ID, uuid.New()); err != nil {
		t.Fatal(err)
	}
}

// equalResult compares upsert params by value, following the pointers.
func equalResult(a, b repos.UpsertWorkoutResultParams) bool {
	eq := func(x, y *int32) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return a.WorkoutID == b.WorkoutID && a.BlockPosition == b.BlockPosition && a.ScoreType == b.ScoreType &&
		eq(a.TimeSeconds, b.TimeSeconds) && eq(a.Rounds, b.Rounds) && eq(a.Reps, b.Reps) && eq(a.Calories, b.Calories) &&
		a.Rx == b.Rx && a.Notes == b.Notes
}
