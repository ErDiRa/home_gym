package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/erdira/homegym/backend/common/errs"
)

// defaultActivityFactor is a day mostly at a desk with some walking.
const defaultActivityFactor = 1.3

// ProfileInput is a user's body as entered; BirthDate is YYYY-MM-DD, an ActivityFactor of 0 the
// default.
type ProfileInput struct {
	HeightCm       float64
	BirthDate      string
	Sex            Sex
	ActivityFactor float64
}

// SaveProfile sets the user's profile, replacing the one before.
func (s *Service) SaveProfile(ctx context.Context, userID uuid.UUID, in ProfileInput) error {
	if in.HeightCm < 100 || in.HeightCm > 250 {
		return errs.Invalid("height must be between 100 and 250 cm")
	}
	birth, err := time.Parse(time.DateOnly, in.BirthDate)
	if err != nil {
		return errs.Invalid("birth_date must be a date (YYYY-MM-DD)")
	}
	if a := age(birth, s.now()); a < 10 || a > 110 {
		return errs.Invalid("birth_date must be between 10 and 110 years ago")
	}
	if in.Sex != Male && in.Sex != Female {
		return errs.Invalid("sex must be male or female")
	}
	if in.ActivityFactor == 0 {
		in.ActivityFactor = defaultActivityFactor
	}
	if in.ActivityFactor < 1 || in.ActivityFactor > 2 {
		return errs.Invalid("activity factor must be between 1.0 and 2.0")
	}
	return s.repo.SaveProfile(ctx, userID, Profile{
		HeightCm: in.HeightCm, BirthDate: birth, Sex: in.Sex, ActivityFactor: in.ActivityFactor,
	})
}

// WeightInput is a weight as entered, with the body fat when measured too; MeasuredOn is
// YYYY-MM-DD.
type WeightInput struct {
	MeasuredOn     string
	WeightKg       float64
	BodyFatPercent *float64
	Notes          string
}

// SaveWeight logs the user's weight (and body fat) of a day, replacing the one logged for that day
// before.
func (s *Service) SaveWeight(ctx context.Context, userID uuid.UUID, in WeightInput) (Weight, error) {
	day, err := s.pastDay(in.MeasuredOn, "measured_on")
	if err != nil {
		return Weight{}, err
	}
	if in.WeightKg < 20 || in.WeightKg > 400 {
		return Weight{}, errs.Invalid("weight must be between 20 and 400 kg")
	}
	if bf := in.BodyFatPercent; bf != nil && (*bf < 3 || *bf > 60) {
		return Weight{}, errs.Invalid("body fat must be between 3 and 60 %")
	}
	notes, err := checkNotes(in.Notes)
	if err != nil {
		return Weight{}, err
	}
	return s.repo.SaveWeight(ctx, userID, Weight{MeasuredOn: day, WeightKg: in.WeightKg, BodyFatPercent: in.BodyFatPercent, Notes: notes})
}

// DeleteWeight removes an entry of the user's weight history.
func (s *Service) DeleteWeight(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.DeleteWeight(ctx, userID, id); errors.Is(err, ErrNotFound) {
		return errs.NotFound("weight not found", err)
	} else if err != nil {
		return err
	}
	return nil
}

// pastDay parses a day the client sent (YYYY-MM-DD, named param in messages) that must not be in
// the future. The client sends its local day, which may be a day ahead of the server's.
func (s *Service) pastDay(value, param string) (time.Time, error) {
	day, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, errs.Invalid(param + " must be a date (YYYY-MM-DD)")
	}
	if day.Year() < 1900 || day.After(s.now().AddDate(0, 0, 1)) {
		return time.Time{}, errs.Invalid(param + " must not be in the future")
	}
	return day, nil
}

// checkNotes trims notes and checks their length.
func checkNotes(notes string) (string, error) {
	notes = strings.TrimSpace(notes)
	if utf8.RuneCountInString(notes) > 1000 {
		return "", errs.Invalid("notes can have at most 1000 characters")
	}
	return notes, nil
}
