package app

import (
	"math"
	"time"
)

// EnergySource tells where a target's energy comes from.
type EnergySource string

const (
	// SourceWhoop is the energy WHOOP measured over the day.
	SourceWhoop EnergySource = "whoop"
	// SourceEstimate is estimated from the body and the day's workouts.
	SourceEstimate EnergySource = "estimate"
)

const (
	kjPerKcal = 4.184
	// proteinPerKg and fatPerKg are grams per kg of bodyweight (protein in the middle of the
	// 1.6-2.2 g/kg recommended for strength training).
	proteinPerKg = 2.0
	fatPerKg     = 0.9
	// minFatShare is the least part of the energy from fat.
	minFatShare = 0.2
	// workoutMET is a moderate strength or CrossFit session; one MET (resting) is in the base
	// already.
	workoutMET = 5.0
	// maxWorkoutHours caps a workout's duration: one left open for hours wasn't trained throughout.
	maxWorkoutHours = 2.5
)

// Target is what a user should eat on a day to keep their weight.
type Target struct {
	Kcal     float64
	ProteinG float64
	FatG     float64
	CarbsG   float64
	Source   EnergySource
	WeightKg float64 // the weight it was computed for
}

// restingEnergy is the energy (kcal) a body spends at rest on a day: Katch-McArdle from the lean
// mass when the body fat is known (bodyFat, nil when not), else Mifflin-St Jeor.
func restingEnergy(p Profile, weightKg float64, bodyFat *float64, day time.Time) float64 {
	if bodyFat != nil {
		return 370 + 21.6*weightKg*(1-*bodyFat/100)
	}
	bmr := 10*weightKg + 6.25*p.HeightCm - 5*float64(age(p.BirthDate, day))
	if p.Sex == Female {
		return bmr - 161
	}
	return bmr + 5
}

// age is the age in whole years on day.
func age(birth, day time.Time) int {
	years := day.Year() - birth.Year()
	if day.Month() < birth.Month() || day.Month() == birth.Month() && day.Day() < birth.Day() {
		years--
	}
	return years
}

// workoutEnergy is the energy (kcal) the workouts add to the day's base: (MET - 1) × kg × hours.
func workoutEnergy(weightKg float64, durations []time.Duration) float64 {
	var hours float64
	for _, d := range durations {
		hours += min(max(d.Hours(), 0), maxWorkoutHours)
	}
	return (workoutMET - 1) * weightKg * hours
}

// target splits the day's energy: protein and fat per kg (fat at least a fifth of the energy),
// the rest carbs. whoopKJ is the energy WHOOP measured over the day, nil for the estimate.
func target(p Profile, weightKg float64, bodyFat *float64, day time.Time, whoopKJ *float64, workouts []time.Duration) Target {
	t := Target{Source: SourceEstimate, WeightKg: weightKg}
	if whoopKJ != nil {
		t.Kcal, t.Source = *whoopKJ/kjPerKcal, SourceWhoop
	} else {
		t.Kcal = restingEnergy(p, weightKg, bodyFat, day)*p.ActivityFactor + workoutEnergy(weightKg, workouts)
	}
	t.Kcal = math.Round(t.Kcal)
	t.ProteinG = math.Round(proteinPerKg * weightKg)
	t.FatG = math.Round(max(fatPerKg*weightKg, minFatShare*t.Kcal/9))
	t.CarbsG = math.Round(max(0, (t.Kcal-4*t.ProteinG-9*t.FatG)/4))
	return t
}
