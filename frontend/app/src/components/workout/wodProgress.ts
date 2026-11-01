// How far a time-capped "for time" WOD got, in the terms of the plan: the full rounds done, the
// next exercise of the round (an index into its lines) and the reps done of that one. The score
// of a capped WOD is the total reps; this turns progress into it and back, so the user doesn't
// add the reps up by hand (18 + 1 + 15 + 2 + … + 4), and shows which lines were finished.

// What the calculation needs of a line of the WOD: its reps, and the unit they are in.
export interface ProgressLine {
  repsValue: number | null // a range counts with its lower value
  repsUnit: string
}

export interface Progress {
  rounds: number // full rounds done
  step: number // the next line of the round, 0-based
  reps: number // reps done of that line, less than its reps
}

export const noProgress: Progress = { rounds: 0, step: 0, reps: 0 }

// Whether progress can be entered: every line is a plain number of reps (not ft, cal, m or s,
// whose amounts don't add up to reps).
export function progressEnabled(lines: ProgressLine[]): boolean {
  return lines.length > 0 && lines.every((l) => l.repsUnit === 'reps' && l.repsValue !== null && l.repsValue > 0)
}

const repsOf = (l: ProgressLine) => l.repsValue ?? 0
const roundReps = (lines: ProgressLine[]) => lines.reduce((sum, l) => sum + repsOf(l), 0)

// Keeps a progress inside the plan: below the block's rounds (all rounds done isn't capped), a
// line of the round, and fewer reps than that line has.
export function clampProgress(lines: ProgressLine[], rounds: number, p: Progress): Progress {
  const whole = (v: unknown) => (typeof v === 'number' && Number.isFinite(v) ? Math.max(0, Math.floor(v)) : 0)
  const step = Math.min(whole(p.step), Math.max(lines.length - 1, 0))
  return {
    rounds: Math.min(whole(p.rounds), Math.max(rounds - 1, 0)),
    step,
    reps: Math.min(whole(p.reps), Math.max(repsOf(lines[step] ?? { repsValue: 0, repsUnit: '' }) - 1, 0)),
  }
}

// The total reps of a progress.
export function progressReps(lines: ProgressLine[], rounds: number, p: Progress): number {
  const c = clampProgress(lines, rounds, p)
  return c.rounds * roundReps(lines) + roundReps(lines.slice(0, c.step)) + c.reps
}

// The progress that makes a total (what a saved score was): the inverse of progressReps. A total
// beyond the plan counts as the last rep of it; one on the border of two lines belongs to the
// next one (0 reps done of it).
export function progressOfReps(lines: ProgressLine[], rounds: number, total: number): Progress {
  const perRound = roundReps(lines)
  if (perRound <= 0 || !Number.isFinite(total) || total <= 0) return { ...noProgress }
  const left = Math.min(Math.floor(total), perRound * rounds - 1)
  let rest = left % perRound
  let step = 0
  while (step < lines.length - 1 && rest >= repsOf(lines[step])) rest -= repsOf(lines[step++])
  return { rounds: Math.floor(left / perRound), step, reps: rest }
}

// How many lines (steps over all rounds) a progress finished.
export const finishedSteps = (lines: ProgressLine[], p: Progress): number => p.rounds * lines.length + p.step
