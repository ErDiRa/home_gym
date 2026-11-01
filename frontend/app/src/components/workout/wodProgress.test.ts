import { describe, expect, it } from 'vitest'
import { clampProgress, finishedSteps, noProgress, progressEnabled, progressOfReps, progressReps, type ProgressLine } from './wodProgress'

const line = (repsValue: number | null, repsUnit = 'reps'): ProgressLine => ({ repsValue, repsUnit })

// The Semi-Finals ladder in one round: hang cleans and rope climbs, 84 reps in all.
const ladder = [18, 1, 15, 2, 12, 3, 9, 4, 6, 5, 3, 6].map((n) => line(n))

describe('progressReps', () => {
  it.each([
    ['nothing done', 0, 0, 0],
    ['reps of the first line', 0, 7, 7],
    ['the first line done', 1, 0, 18],
    ['eight lines done: 18+1+15+2+12+3+9+4', 8, 0, 64],
    ['the WODup example: 4 of 6 hang cleans on top', 8, 4, 68],
    ['eleven lines (78 reps) and 5 of the last 6', 11, 5, 83],
  ])('%s', (_, step, reps, total) => {
    expect(progressReps(ladder, 1, { rounds: 0, step, reps })).toBe(total)
  })

  it('counts the full rounds of a WOD with several: 130 reps a round', () => {
    const rounds3 = [line(30), line(100)]
    expect(progressReps(rounds3, 3, { rounds: 1, step: 1, reps: 12 })).toBe(130 + 30 + 12)
    expect(progressReps(rounds3, 3, { rounds: 2, step: 0, reps: 0 })).toBe(260)
  })

  it('counts a range with its lower value', () => {
    expect(progressReps([line(8), line(5)], 1, { rounds: 0, step: 1, reps: 2 })).toBe(10)
  })
})

describe('progressOfReps', () => {
  it('turns the total back into rounds, line and reps', () => {
    expect(progressOfReps(ladder, 1, 68)).toEqual({ rounds: 0, step: 8, reps: 4 })
    expect(progressOfReps([line(30), line(100)], 3, 172)).toEqual({ rounds: 1, step: 1, reps: 12 })
  })

  it('puts a total on the border of two lines at the start of the next', () => {
    expect(progressOfReps(ladder, 1, 64)).toEqual({ rounds: 0, step: 8, reps: 0 })
    expect(progressOfReps(ladder, 1, 18)).toEqual({ rounds: 0, step: 1, reps: 0 })
    expect(progressOfReps([line(30), line(100)], 3, 130)).toEqual({ rounds: 1, step: 0, reps: 0 })
  })

  it('is no progress for 0 reps, and stops at the last rep of the plan', () => {
    expect(progressOfReps(ladder, 1, 0)).toEqual(noProgress)
    expect(progressOfReps(ladder, 1, 84)).toEqual({ rounds: 0, step: 11, reps: 5 })
    expect(progressOfReps(ladder, 1, 500)).toEqual({ rounds: 0, step: 11, reps: 5 })
  })

  it('gives back what progressReps counted, for every place in the ladder', () => {
    for (const [step, l] of ladder.entries()) {
      for (let reps = 0; reps < l.repsValue!; reps++) {
        const p = { rounds: 0, step, reps }
        expect(progressOfReps(ladder, 1, progressReps(ladder, 1, p))).toEqual(p)
      }
    }
  })

  it('does the same over several rounds', () => {
    const lines = [line(30), line(100)]
    for (let rounds = 0; rounds < 3; rounds++) {
      for (const [step, l] of lines.entries()) {
        const p = { rounds, step, reps: l.repsValue! - 1 }
        expect(progressOfReps(lines, 3, progressReps(lines, 3, p))).toEqual(p)
      }
    }
  })
})

describe('the inputs', () => {
  it('are only for lines of plain reps', () => {
    expect(progressEnabled(ladder)).toBe(true)
    expect(progressEnabled([line(30), line(75, 'ft')])).toBe(false)
    expect(progressEnabled([line(30), line(null)])).toBe(false)
    expect(progressEnabled([])).toBe(false)
  })

  it('stay inside the plan', () => {
    // 6 hang cleans in that line: at most 5 reps; one round of one: no full rounds.
    expect(clampProgress(ladder, 1, { rounds: 4, step: 40, reps: 99 })).toEqual({ rounds: 0, step: 11, reps: 5 })
    expect(clampProgress(ladder, 1, { rounds: -1, step: -2, reps: -3 })).toEqual(noProgress)
    expect(clampProgress([line(30), line(100)], 3, { rounds: 7, step: 0, reps: 0 }).rounds).toBe(2)
    expect(progressReps(ladder, 1, { rounds: 0, step: 8, reps: 99 })).toBe(64 + 5)
  })

  it('count the finished lines over all rounds', () => {
    expect(finishedSteps(ladder, { rounds: 0, step: 8, reps: 4 })).toBe(8)
    expect(finishedSteps([line(30), line(100)], { rounds: 1, step: 1, reps: 12 })).toBe(3)
  })
})
