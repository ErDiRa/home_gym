package whoop

import (
	"context"
	"errors"
	"math"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/whoop/repos"
)

// notFoundActivity is WHOOP not knowing the activity (or it being another member's).
var notFoundActivity = service.Invalid("unknown WHOOP activity")

// activity is a WHOOP workout as the API returns it (only the fields used).
type activity struct {
	ID         uuid.UUID `json:"id"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	SportName  string    `json:"sport_name"`
	ScoreState string    `json:"score_state"` // SCORED, PENDING_SCORE or UNSCORABLE
	Score      *struct {
		Strain           float64 `json:"strain"`
		AverageHeartRate int32   `json:"average_heart_rate"`
		MaxHeartRate     int32   `json:"max_heart_rate"`
		Kilojoule        float64 `json:"kilojoule"`
		ZoneDurations    struct {
			Zero  int32 `json:"zone_zero_milli"`
			One   int32 `json:"zone_one_milli"`
			Two   int32 `json:"zone_two_milli"`
			Three int32 `json:"zone_three_milli"`
			Four  int32 `json:"zone_four_milli"`
			Five  int32 `json:"zone_five_milli"`
		} `json:"zone_durations"`
	} `json:"score"`
}

// recovery is a WHOOP recovery as the API returns it (only the fields used).
type recovery struct {
	Score *struct {
		RecoveryScore    float64 `json:"recovery_score"`
		RestingHeartRate float64 `json:"resting_heart_rate"`
		HrvRmssdMilli    float64 `json:"hrv_rmssd_milli"`
	} `json:"score"`
}

// Candidate is a WHOOP activity the user can link to a workout.
type Candidate struct {
	ID        uuid.UUID `json:"id"`
	SportName string    `json:"sport_name"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Strain    *float64  `json:"strain"`
	AvgHR     *int32    `json:"avg_hr"`
}

// Candidates lists the user's WHOOP activities that started in [from, to) (the client's local
// day of the workout), newest first, for picking the one to link to the workout.
func (s *Service) Candidates(ctx context.Context, userID, workoutID uuid.UUID, from, to time.Time) ([]Candidate, error) {
	if from.IsZero() || to.IsZero() || !from.Before(to) || to.Sub(from) > 48*time.Hour {
		return nil, service.Invalid("from and to are required (RFC 3339), from before to, at most two days apart")
	}
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return nil, err
	}
	var page struct {
		Records []activity `json:"records"`
	}
	query := url.Values{"start": {from.Format(time.RFC3339)}, "end": {to.Format(time.RFC3339)}, "limit": {"25"}}
	if err := s.get(ctx, userID, "/developer/v2/activity/workout", query, &page); err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(page.Records))
	for _, a := range page.Records {
		c := Candidate{ID: a.ID, SportName: a.SportName, Start: a.Start, End: a.End}
		if a.Score != nil {
			c.Strain, c.AvgHR = &a.Score.Strain, &a.Score.AverageHeartRate
		}
		out = append(out, c)
	}
	return out, nil
}

// Link fetches a WHOOP activity and the recovery of the day before it started and stores them
// with one of the user's workouts. Linking again replaces them, e.g. once WHOOP has scored the
// activity.
func (s *Service) Link(ctx context.Context, userID, workoutID, whoopID uuid.UUID) (repos.WorkoutWhoopDatum, error) {
	if err := s.checkWorkout(ctx, userID, workoutID); err != nil {
		return repos.WorkoutWhoopDatum{}, err
	}
	var a activity
	if err := s.get(ctx, userID, "/developer/v2/activity/workout/"+whoopID.String(), nil, &a); err != nil {
		return repos.WorkoutWhoopDatum{}, err
	}
	// The recovery is scored on waking, so the newest one in the day before the activity.
	var recoveries struct {
		Records []recovery `json:"records"`
	}
	query := url.Values{"start": {a.Start.Add(-24 * time.Hour).Format(time.RFC3339)}, "end": {a.Start.Format(time.RFC3339)}, "limit": {"1"}}
	if err := s.get(ctx, userID, "/developer/v2/recovery", query, &recoveries); err != nil {
		return repos.WorkoutWhoopDatum{}, err
	}

	arg := repos.SaveWorkoutDataParams{
		WorkoutID: workoutID, WhoopWorkoutID: a.ID, SportName: a.SportName, ScoreState: a.ScoreState,
		StartAt: pgtype.Timestamptz{Time: a.Start, Valid: true}, EndAt: pgtype.Timestamptz{Time: a.End, Valid: true},
	}
	if sc := a.Score; sc != nil {
		z := sc.ZoneDurations
		arg.Strain, arg.AvgHr, arg.MaxHr, arg.Kilojoule = &sc.Strain, &sc.AverageHeartRate, &sc.MaxHeartRate, &sc.Kilojoule
		arg.ZoneMs = []int32{z.Zero, z.One, z.Two, z.Three, z.Four, z.Five}
	}
	if len(recoveries.Records) > 0 && recoveries.Records[0].Score != nil {
		r := recoveries.Records[0].Score
		score, rhr := int32(math.Round(r.RecoveryScore)), int32(math.Round(r.RestingHeartRate))
		arg.RecoveryScore, arg.RestingHr, arg.HrvMs = &score, &rhr, &r.HrvRmssdMilli
	}
	return s.q.SaveWorkoutData(ctx, arg)
}

// Get returns the WHOOP data linked to one of the user's workouts, nil when there is none.
func (s *Service) Get(ctx context.Context, userID, workoutID uuid.UUID) (*repos.WorkoutWhoopDatum, error) {
	d, err := s.q.GetWorkoutData(ctx, repos.GetWorkoutDataParams{WorkoutID: workoutID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Unlink removes the WHOOP data from one of the user's workouts.
func (s *Service) Unlink(ctx context.Context, userID, workoutID uuid.UUID) error {
	n, err := s.q.DeleteWorkoutData(ctx, repos.DeleteWorkoutDataParams{WorkoutID: workoutID, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("no WHOOP data linked", nil)
	}
	return nil
}
