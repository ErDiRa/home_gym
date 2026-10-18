package training

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/erdira/homegym/backend/internal/service"
	"github.com/erdira/homegym/backend/internal/service/training/repos"
)

// validate trims and validates a training against the lookups, in place. It returns a
// service.Invalid error for the first problem.
func validate(in *Input, lk lookups) error {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if in.Name == "" || tooLong(in.Name, 100) {
		return service.Invalid("name is required (max 100 characters)")
	}
	start, startErr := time.Parse(time.DateOnly, in.StartDate)
	end, endErr := time.Parse(time.DateOnly, in.EndDate)
	if startErr != nil || endErr != nil {
		return service.Invalid("start_date and end_date are required (YYYY-MM-DD)")
	}
	if end.Before(start) {
		return service.Invalid("end_date must not be before start_date")
	}
	in.start, in.end = pgtype.Date{Time: start, Valid: true}, pgtype.Date{Time: end, Valid: true}
	if in.SessionsPerWeek < 1 || in.SessionsPerWeek > 14 {
		return service.Invalid("sessions_per_week must be 1-14")
	}
	// A unit id sent twice would update the same row twice and lose one of the units.
	seenUnits := map[uuid.UUID]bool{}
	for i := range in.Units {
		u := &in.Units[i]
		u.Name, u.Notes = strings.TrimSpace(u.Name), strings.TrimSpace(u.Notes)
		if u.Name == "" || tooLong(u.Name, 100) {
			return service.Invalid("every unit needs a name (max 100 characters)")
		}
		if u.ID != nil {
			if seenUnits[*u.ID] {
				return service.Invalid("a unit appears twice")
			}
			seenUnits[*u.ID] = true
		}
		for j := range u.Blocks {
			if msg := checkBlock(&u.Blocks[j], lk); msg != "" {
				return service.Invalid(fmt.Sprintf("%s, block %s: %s", u.Name, blockLetter(j), msg))
			}
		}
	}
	return nil
}

// checkBlock trims and validates a block against its kind, clearing sets, reps and load on the
// side (block or exercises) the kind doesn't use. It returns a message for the first problem.
func checkBlock(b *BlockInput, lk lookups) string {
	b.Name, b.Notes = strings.TrimSpace(b.Name), strings.TrimSpace(b.Notes)
	kind, ok := lk.kinds[b.Kind]
	if !ok {
		return "unknown block kind"
	}
	if tooLong(b.Name, 100) {
		return "the name can have at most 100 characters"
	}
	n := int32(len(b.Exercises))
	if n < kind.MinExercises || (kind.MaxExercises != nil && n > *kind.MaxExercises) {
		return exerciseCountRule(kind)
	}

	// What the kind shares is validated on the block; the rest is cleared there and validated
	// per exercise below.
	if kind.SharedSets {
		if !validSets(b.Sets) {
			return "sets must be 1-100"
		}
	} else {
		b.Sets = nil
	}

	if kind.SharedReps {
		if msg := checkReps(&b.Amount, lk); msg != "" {
			return msg
		}
	} else {
		b.clearReps()
	}
	if kind.SharedLoad {
		if msg := checkLoad(&b.Amount, lk); msg != "" {
			return msg
		}
	} else {
		b.clearLoad()
	}
	for k := range b.Exercises {
		e := &b.Exercises[k]
		e.Tempo, e.Notes = strings.TrimSpace(e.Tempo), strings.TrimSpace(e.Notes)
		if e.ExerciseID == uuid.Nil {
			return "every exercise needs an exercise_id"
		}
		if tooLong(e.Tempo, 20) {
			return fmt.Sprintf("exercise %d: the tempo can have at most 20 characters", k+1)
		}
		if kind.SharedSets {
			e.Sets = nil
		} else if !validSets(e.Sets) {
			return fmt.Sprintf("exercise %d: sets must be 1-100", k+1)
		}
		if kind.SharedReps {
			e.clearReps()
		} else if msg := checkReps(&e.Amount, lk); msg != "" {
			return fmt.Sprintf("exercise %d: %s", k+1, msg)
		}
		if kind.SharedLoad {
			e.clearLoad()
		} else if msg := checkLoad(&e.Amount, lk); msg != "" {
			return fmt.Sprintf("exercise %d: %s", k+1, msg)
		}
	}
	return ""
}

// validSets reports whether sets is set and within the column's CHECK (1-100).
func validSets(sets *int32) bool { return sets != nil && *sets >= 1 && *sets <= 100 }

// exerciseCountRule describes how many exercises a kind of block takes.
func exerciseCountRule(kind repos.ListBlockKindsRow) string {
	switch {
	case kind.MaxExercises == nil:
		return fmt.Sprintf("a %s block needs at least %d %s", kind.Name, kind.MinExercises, plural(kind.MinExercises, "exercise"))
	case *kind.MaxExercises == kind.MinExercises:
		return fmt.Sprintf("a %s block takes exactly %d %s", kind.Name, kind.MinExercises, plural(kind.MinExercises, "exercise"))
	default:
		return fmt.Sprintf("a %s block takes %d to %d exercises", kind.Name, kind.MinExercises, *kind.MaxExercises)
	}
}

// plural adds an "s" to word unless n is 1.
func plural(n int32, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// checkReps validates the required reps of an amount.
func checkReps(a *Amount, lk lookups) string {
	if a.RepsValue == nil {
		return "reps are required"
	}
	return checkRange("reps", a.RepsValue, a.RepsMax, a.RepsUnit, lk.repUnits)
}

// checkLoad validates the optional load of an amount.
func checkLoad(a *Amount, lk lookups) string {
	if a.LoadValue == nil && a.LoadMax == nil && a.LoadUnit == nil {
		return "" // no load, e.g. bodyweight
	}
	return checkRange("load", a.LoadValue, a.LoadMax, a.LoadUnit, lk.loadUnits)
}

// checkRange validates a value with an optional max in one of units.
func checkRange(what string, value, max *float64, unitID *uuid.UUID, units map[uuid.UUID]unit) string {
	if value == nil || unitID == nil {
		return what + " needs a value and a unit"
	}
	u, ok := units[*unitID]
	if !ok {
		return "unknown " + what + " unit"
	}
	if max != nil && *max <= *value {
		return what + ": the upper end of a range must be higher than the value"
	}
	for _, v := range []*float64{value, max} {
		if v == nil {
			continue
		}
		// Within the column's range and within the unit's own bounds (e.g. RPE 1-10).
		if *v < 0 || *v >= maxAmount || (u.min != nil && *v < *u.min) || (u.max != nil && *v > *u.max) {
			return fmt.Sprintf("%s in %s must be %s", what, u.name, boundsText(u))
		}
	}
	return ""
}

// boundsText describes the values a unit allows, e.g. "between 1 and 10".
func boundsText(u unit) string {
	lo, hi := 0.0, float64(maxAmount-1)
	if u.min != nil {
		lo = *u.min
	}
	if u.max != nil {
		hi = *u.max
	}
	return fmt.Sprintf("between %g and %g", lo, hi)
}

// blockLetter labels a block by its position in the unit: A, B, ... Z, then 27, 28...
func blockLetter(i int) string {
	if i < 26 {
		return string(rune('A' + i))
	}
	return strconv.Itoa(i + 1)
}
