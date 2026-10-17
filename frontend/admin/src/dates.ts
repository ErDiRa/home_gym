// Calendar dates travel as YYYY-MM-DD strings. They are handled as local dates, so they don't
// shift by a day in time zones west of UTC.

function parse(date: string): Date {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y, m - 1, d)
}

function toIso(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function addDays(date: string, days: number): string {
  const d = parse(date)
  d.setDate(d.getDate() + days)
  return toIso(d)
}

// Whole days from one date to another; rounded, as a DST change makes a day 23 or 25 hours.
export function daysBetween(from: string, to: string): number {
  return Math.round((parse(to).getTime() - parse(from).getTime()) / 86_400_000)
}

// The coming Monday, or today when it is one: when a new training block usually starts.
export function nextMonday(): string {
  const today = new Date()
  const ahead = (8 - today.getDay()) % 7 // getDay: 0 = Sunday, 1 = Monday
  today.setDate(today.getDate() + ahead)
  return toIso(today)
}

export function formatDate(date: string, options?: Intl.DateTimeFormatOptions): string {
  return parse(date).toLocaleDateString(undefined, options)
}

// A training period, e.g. "12 Oct – 8 Nov 2026".
export function formatPeriod(start: string, end: string): string {
  const sameYear = start.slice(0, 4) === end.slice(0, 4)
  const from = formatDate(start, sameYear ? { day: 'numeric', month: 'short' } : { dateStyle: 'medium' })
  return `${from} – ${formatDate(end, { dateStyle: 'medium' })}`
}
