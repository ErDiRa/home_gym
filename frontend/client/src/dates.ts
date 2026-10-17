// Calendar dates travel as YYYY-MM-DD strings. They are handled as local dates, so they don't
// shift by a day in time zones west of UTC; as strings they compare and sort in date order.

function parse(date: string): Date {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y, m - 1, d)
}

function toIso(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function today(): string {
  return toIso(new Date())
}

export function addDays(date: string, days: number): string {
  const d = parse(date)
  d.setDate(d.getDate() + days)
  return toIso(d)
}

// ISO weekday, 1 = Monday to 7 = Sunday (getDay counts from Sunday = 0).
export function isoWeekday(date: string): number {
  return ((parse(date).getDay() + 6) % 7) + 1
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
