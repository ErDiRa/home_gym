import type { ResultInput, Score, ScoreType } from '../../types'

// The formats a WOD block is scored in, like the server's (session_blocks.score_type). Each says
// what its time means (a cap or the duration, required or not), whether it has an interval
// (EMOM), and what its result is made of.
export interface ScoreFormat {
  code: ScoreType
  label: string
  description: string
  timeLabel: 'Time cap' | 'Duration'
  timeRequired: boolean
  hasInterval: boolean
}

export const scoreFormats: ScoreFormat[] = [
  {
    code: 'for_time',
    label: 'For time',
    description: 'As fast as possible; the score is the time, or the reps done when the cap is hit',
    timeLabel: 'Time cap',
    timeRequired: false,
    hasInterval: false,
  },
  {
    code: 'amrap',
    label: 'AMRAP',
    description: 'As many rounds and reps as possible in the time',
    timeLabel: 'Duration',
    timeRequired: true,
    hasInterval: false,
  },
  {
    code: 'max_reps',
    label: 'Max reps',
    description: 'As many reps as possible, e.g. max burpees in 5 minutes',
    timeLabel: 'Duration',
    timeRequired: false,
    hasInterval: false,
  },
  {
    code: 'max_calories',
    label: 'Max cal',
    description: 'As many calories as possible, e.g. on the rower or bike',
    timeLabel: 'Duration',
    timeRequired: false,
    hasInterval: false,
  },
  {
    code: 'emom',
    label: 'EMOM',
    description: 'Every minute (or interval) on the minute; the score is the rounds completed',
    timeLabel: 'Duration',
    timeRequired: true,
    hasInterval: true,
  },
]

export const scoreFormat = (code: ScoreType | null) => scoreFormats.find((f) => f.code === code)

// The interval of an EMOM unless set, like the server's default.
export const DEFAULT_INTERVAL = 60
// At most 4 hours for a cap, duration or interval, like the server's maxScoreTime.
export const MAX_SCORE_TIME = 4 * 60 * 60

// Seconds as a clock: 0:45, 12:00, 1:05:30.
export function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  const ss = String(s).padStart(2, '0')
  return h ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${m}:${ss}`
}

// How a WOD block is planned, e.g. "AMRAP 12:00", "For time · cap 10:00 · 3 rounds",
// "EMOM 10:00 · every 1:00".
export function wodSummary(score: Score, rounds: number | null): string {
  const format = scoreFormat(score.score_type)
  if (!format) return ''
  const parts = [format.label]
  const time = score.time_seconds
  if (time !== null) {
    if (format.timeLabel === 'Time cap') parts.push(`cap ${formatDuration(time)}`)
    else parts[0] += ` ${formatDuration(time)}`
  }
  if (format.hasInterval && score.interval_seconds !== null) parts.push(`every ${formatDuration(score.interval_seconds)}`)
  if (rounds) parts.push(`${rounds} ${rounds === 1 ? 'round' : 'rounds'}`)
  return parts.join(' · ')
}

// The EMOM's number of intervals, e.g. 10 for 10:00 every 1:00; null when unknown.
export function emomRounds(score: Score): number | null {
  if (score.score_type !== 'emom' || !score.time_seconds) return null
  return Math.floor(score.time_seconds / (score.interval_seconds || DEFAULT_INTERVAL))
}

// A logged score as text: "8:42", "Capped · 81 reps", "14 rounds + 7 reps", "31 reps", "48 cal",
// "9 rounds" (or "9/10 rounds" for an EMOM of known length). Rx or scaled is shown apart.
export function scoreText(r: Pick<ResultInput, 'score_type' | 'time_seconds' | 'rounds' | 'reps' | 'calories'>, of?: number | null) {
  const n = (v: number | null, unit: string) => `${v ?? 0} ${unit}${v === 1 ? '' : 's'}`
  switch (r.score_type) {
    case 'for_time':
      return r.time_seconds !== null ? formatDuration(r.time_seconds) : `Capped · ${n(r.reps, 'rep')}`
    case 'amrap':
      return r.reps ? `${n(r.rounds, 'round')} + ${n(r.reps, 'rep')}` : n(r.rounds, 'round')
    case 'max_reps':
      return n(r.reps, 'rep')
    case 'max_calories':
      return `${r.calories ?? 0} cal`
    case 'emom':
      return of ? `${r.rounds ?? 0}/${of} rounds` : n(r.rounds, 'round')
  }
}

// Why a WOD block's score can't be saved yet, if anything; mirrors the server's checkScore.
export function scoreProblem(score: Score): string {
  const format = scoreFormat(score.score_type)
  if (!format) return 'Choose how the WOD is scored.'
  const time = score.time_seconds
  if (time === null) {
    if (format.timeRequired) return `${format.label === 'AMRAP' ? 'An AMRAP' : 'An EMOM'} needs a duration.`
  } else if (time < 1 || time > MAX_SCORE_TIME) return 'The time must be between 0:01 and 4:00:00.'
  if (format.hasInterval && time !== null) {
    const interval = score.interval_seconds ?? DEFAULT_INTERVAL
    if (interval < 1 || interval > time) return 'The interval must be at least a second and fit into the duration.'
  }
  return ''
}
