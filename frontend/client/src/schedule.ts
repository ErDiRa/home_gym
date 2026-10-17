import { addDays, isoWeekday } from './dates'
import type { Training, TrainingUnit } from './types'

// A day with training: the units done that day, in the training's order.
export interface Session {
  date: string // YYYY-MM-DD
  units: TrainingUnit[]
}

// The days of a training's period it is done on: each day that falls on one of a unit's
// weekdays or is one of its dates, in order.
export function sessions(training: Training): Session[] {
  const out: Session[] = []
  for (let date = training.start_date; date <= training.end_date; date = addDays(date, 1)) {
    const weekday = isoWeekday(date)
    const units = training.units.filter((u) => u.weekdays.includes(weekday) || u.dates.includes(date))
    if (units.length) out.push({ date, units })
  }
  return out
}
