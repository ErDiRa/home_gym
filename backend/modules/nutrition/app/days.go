package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
	tracking "github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

// maxDays is the longest period Days returns.
const maxDays = 366

// maxGrams is the most of one macro a day can have, as in the nutrition.daily_macros CHECKs.
const maxGrams = 2000

// Day is a user's day: the macros eaten (nil when none were logged), the target (nil without a
// profile or a weight) and the workouts finished that day (local time).
type Day struct {
	Date     time.Time
	Macros   *Macros
	Target   *Target
	Workouts []tracking.FinishedWorkout
}

// Days returns the user's days from from to to (YYYY-MM-DD, both included, at most a year) in the
// client's time zone tz (an IANA name such as Europe/Zurich), which decides the day a workout or
// the WHOOP energy belongs to.
func (s *Service) Days(ctx context.Context, userID uuid.UUID, from, to, tz string) ([]Day, error) {
	loc, err := time.LoadLocation(tz)
	if tz == "" || err != nil {
		return nil, errs.Invalid("tz must be a time zone such as Europe/Zurich")
	}
	first, err1 := time.ParseInLocation(time.DateOnly, from, loc)
	last, err2 := time.ParseInLocation(time.DateOnly, to, loc)
	if err1 != nil || err2 != nil {
		return nil, errs.Invalid("from and to must be dates (YYYY-MM-DD)")
	}
	if last.Before(first) || last.Sub(first) > maxDays*24*time.Hour {
		return nil, errs.Invalid("to must not be before from, at most a year later")
	}
	// The instants bounding the local days.
	start, end := first, last.AddDate(0, 0, 1)

	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	weights, err := s.repo.Weights(ctx, userID)
	if err != nil {
		return nil, err
	}
	// The repository's days are dates at UTC midnight.
	macros, err := s.repo.Macros(ctx, userID, utcDay(first), utcDay(last))
	if err != nil {
		return nil, err
	}
	workouts, err := s.tracking.FinishedWorkouts(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	energy, err := s.whoop.DailyEnergy(ctx, userID, start, end, loc)
	if err != nil {
		return nil, err
	}

	macrosOf := map[string]Macros{}
	for _, m := range macros {
		macrosOf[m.Day.Format(time.DateOnly)] = m
	}
	workoutsOf := map[string][]tracking.FinishedWorkout{}
	for _, w := range workouts {
		key := w.CompletedAt.In(loc).Format(time.DateOnly)
		workoutsOf[key] = append(workoutsOf[key], w)
	}

	var out []Day
	for d := first; d.Before(end); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		day := Day{Date: utcDay(d), Workouts: workoutsOf[key]}
		if m, ok := macrosOf[key]; ok {
			day.Macros = &m
		}
		if weight, ok := weightOn(weights, utcDay(d)); ok && profile != nil {
			var kj *float64
			if e, ok := energy[key]; ok {
				kj = &e
			}
			durations := make([]time.Duration, len(day.Workouts))
			for i, w := range day.Workouts {
				durations[i] = w.CompletedAt.Sub(w.StartedAt)
			}
			t := target(*profile, weight, bodyFatOn(weights, utcDay(d)), d, kj, durations)
			day.Target = &t
		}
		out = append(out, day)
	}
	return out, nil
}

// weightOn is the weight of a day: the newest measured on or before it, else (a day before the
// first entry) the first one. weights are newest first.
func weightOn(weights []Weight, day time.Time) (float64, bool) {
	for _, w := range weights {
		if !w.MeasuredOn.After(day) {
			return w.WeightKg, true
		}
	}
	if len(weights) == 0 {
		return 0, false
	}
	return weights[len(weights)-1].WeightKg, true
}

// bodyFatOn is the body fat of a day: the newest measured on or before it; nil without one, as it
// is optional. weights are newest first.
func bodyFatOn(weights []Weight, day time.Time) *float64 {
	for _, w := range weights {
		if w.BodyFatPercent != nil && !w.MeasuredOn.After(day) {
			return w.BodyFatPercent
		}
	}
	return nil
}

// utcDay is the date of t as a time at UTC midnight, as dates are stored.
func utcDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// MacrosInput are a day's totals as entered: each may be left out, not all.
type MacrosInput struct {
	ProteinG *float64
	FatG     *float64
	CarbsG   *float64
	Notes    string
}

// SaveMacros logs the macros the user ate on a day (YYYY-MM-DD), replacing the ones logged for it
// before.
func (s *Service) SaveMacros(ctx context.Context, userID uuid.UUID, date string, in MacrosInput) (Macros, error) {
	day, err := s.pastDay(date, "date")
	if err != nil {
		return Macros{}, err
	}
	if in.ProteinG == nil && in.FatG == nil && in.CarbsG == nil {
		return Macros{}, errs.Invalid("enter protein, fat or carbs")
	}
	for _, g := range []*float64{in.ProteinG, in.FatG, in.CarbsG} {
		if g != nil && (*g < 0 || *g > maxGrams) {
			return Macros{}, errs.Invalid("grams must be between 0 and 2000")
		}
	}
	notes, err := checkNotes(in.Notes)
	if err != nil {
		return Macros{}, err
	}
	return s.repo.SaveMacros(ctx, userID, Macros{Day: day, ProteinG: in.ProteinG, FatG: in.FatG, CarbsG: in.CarbsG, Notes: notes})
}

// DeleteMacros removes the macros logged for a day (YYYY-MM-DD).
func (s *Service) DeleteMacros(ctx context.Context, userID uuid.UUID, date string) error {
	day, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return errs.Invalid("date must be a date (YYYY-MM-DD)")
	}
	if err := s.repo.DeleteMacros(ctx, userID, day); errors.Is(err, ErrNotFound) {
		return errs.NotFound("no macros logged that day", err)
	} else if err != nil {
		return err
	}
	return nil
}
