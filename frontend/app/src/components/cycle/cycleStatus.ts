import { formatDate, today, weekOf } from '../../dates'

// Where today is in a cycle's weeks: "Week 3 of 10", "Starts 26 Oct 2026" or "Ended".
export function cycleStatus(c: { start_date: string; end_date: string; weeks: number }, date = today()): string {
  const week = weekOf(c.start_date, c.weeks, date)
  if (week !== null) return `Week ${week} of ${c.weeks}`
  return date < c.start_date ? `Starts ${formatDate(c.start_date)}` : 'Ended'
}

// Whether a training runs wholly within a cycle's weeks.
export const withinCycle = (t: { start_date: string; end_date: string }, c: { start_date: string; end_date: string }) =>
  t.start_date >= c.start_date && t.end_date <= c.end_date
