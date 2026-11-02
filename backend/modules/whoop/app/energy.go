package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// DailyEnergy returns the energy (kJ) WHOOP measured on the user's local days from from (a day's
// start in loc) up to to, keyed by the day (YYYY-MM-DD). A day gets the finished, scored cycle that
// overlaps it most (a cycle runs from falling asleep to falling asleep again), so the current day
// has none yet. It is empty without a connection; WHOOP failing (or a connection made before the
// cycles were asked for) is logged and also leaves it empty, as the energy is only an extra.
func (s *Service) DailyEnergy(ctx context.Context, userID uuid.UUID, from, to time.Time, loc *time.Location) (map[string]float64, error) {
	if _, err := s.connections.Connection(ctx, userID); errors.Is(err, ErrNotConnected) {
		return map[string]float64{}, nil
	} else if err != nil {
		return nil, err
	}
	token, err := s.accessToken(ctx, userID)
	if err == nil {
		var cycles []Cycle
		// A cycle starts the evening before the day it mostly covers.
		if cycles, err = s.api.Cycles(ctx, token, from.Add(-24*time.Hour), to); err == nil {
			return energyByDay(cycles, from, to, loc), nil
		}
	}
	slog.WarnContext(ctx, "read WHOOP cycles", "user_id", userID, "error", err)
	return map[string]float64{}, nil
}

// energyByDay assigns each finished, scored cycle to the local day in [from, to) it overlaps most.
func energyByDay(cycles []Cycle, from, to time.Time, loc *time.Location) map[string]float64 {
	out := map[string]float64{}
	best := map[string]time.Duration{} // the overlap of the cycle a day has
	for _, c := range cycles {
		if c.End == nil || c.Kilojoule == nil || c.ScoreState != "SCORED" {
			continue
		}
		var day string
		var most time.Duration
		start := c.Start.In(loc)
		for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc); d.Before(*c.End); d = d.AddDate(0, 0, 1) {
			next := d.AddDate(0, 0, 1)
			if overlap := minTime(next, *c.End).Sub(maxTime(d, c.Start)); overlap > most {
				day, most = d.Format(time.DateOnly), overlap
			}
		}
		if day == "" || most <= best[day] {
			continue
		}
		if d, _ := time.ParseInLocation(time.DateOnly, day, loc); d.Before(from) || !d.Before(to) {
			continue
		}
		out[day], best[day] = *c.Kilojoule, most
	}
	return out
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
