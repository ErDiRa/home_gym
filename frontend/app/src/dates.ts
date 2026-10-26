// Calendar dates travel as YYYY-MM-DD strings. They are handled as local dates, so they don't
// shift by a day in time zones west of UTC.

function parse(date: string): Date {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y, m - 1, d)
}

// The local start (00:00) of a YYYY-MM-DD date.
export function startOf(date: string): Date {
  return parse(date)
}

export function toIso(d: Date): string {
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

// Whole days from one date to another; rounded, as a DST change makes a day 23 or 25 hours.
export function daysBetween(from: string, to: string): number {
  return Math.round((parse(to).getTime() - parse(from).getTime()) / 86_400_000)
}

// Whether a YYYY-MM-DD date is a Monday, the day trainings start on.
export function isMonday(date: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(date) && parse(date).getDay() === 1
}

// The week of a period starting on a Monday that a date falls in, from 1; null outside its weeks.
export function weekOf(start: string, weeks: number, date: string): number | null {
  const days = daysBetween(start, date)
  const week = Math.floor(days / 7) + 1
  return days >= 0 && week <= weeks ? week : null
}

// The Monday of a date's week.
export function mondayOf(date: string): string {
  return addDays(date, -((parse(date).getDay() + 6) % 7)) // getDay: 0 = Sunday
}

// The coming Monday, or today when it is one: when a new training block usually starts.
export function nextMonday(): string {
  const today = new Date()
  const ahead = (8 - today.getDay()) % 7 // getDay: 0 = Sunday, 1 = Monday
  today.setDate(today.getDate() + ahead)
  return toIso(today)
}

// Dates show in English whatever the browser's language, as day, 3-letter month and year, e.g.
// "19 Oct 2026" (Intl's en-GB would write "Sept"), optionally with the weekday ("Mon, 19 Oct 2026").
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

export interface DateFormat {
  weekday?: boolean
  /** false leaves out the year, e.g. for the days of a week shown next to its training. */
  year?: boolean
}

function format(d: Date, { weekday = false, year = true }: DateFormat = {}): string {
  const day = `${d.getDate()} ${MONTHS[d.getMonth()]}${year ? ` ${d.getFullYear()}` : ''}`
  return weekday ? `${WEEKDAYS[d.getDay()]}, ${day}` : day
}

// A YYYY-MM-DD date, e.g. "19 Oct 2026".
export function formatDate(date: string, options?: DateFormat): string {
  return format(parse(date), options)
}

// A training period, e.g. "19 Oct 2026 – 15 Nov 2026".
export function formatPeriod(start: string, end: string): string {
  return `${formatDate(start)} – ${formatDate(end)}`
}

// The local start (Monday, 00:00) of the week that is `offset` weeks from the current one.
export function weekStart(offset = 0): Date {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7) + offset * 7) // getDay: 0 = Sunday
  return d
}

// The local start of the week after the one starting at start (7 days later, across DST).
export function nextWeek(start: Date): Date {
  const d = new Date(start)
  d.setDate(d.getDate() + 7)
  return d
}

// A moment as its local day, e.g. "Mon, 19 Oct 2026" (weekday: false for "19 Oct 2026").
export function formatDay(at: string | Date, { weekday = true }: { weekday?: boolean } = {}): string {
  return format(new Date(at), { weekday })
}

// A moment's local time of day, 24-hour, e.g. "08:41".
export function formatTime(at: string | Date): string {
  const d = new Date(at)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}
