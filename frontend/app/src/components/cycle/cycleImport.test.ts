import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { beforeEach, describe, expect, it } from 'vitest'
import cycleFile from './__fixtures__/cycle.json?raw'
import trainingWeek from '../training/__fixtures__/training-week.json?raw'
import { byName, useTestLookups } from '../training/__fixtures__/lookups'
import { mergedNewExercises, parseAnyImport, parseCycleImport, toCycleImportInput, trainingStart } from './cycleImport'
import { importErrorMessage, nameKey, unresolvedNames } from '../training/trainingImport'

let kinds: string[]
beforeEach(() => {
  kinds = useTestLookups()
})

function errorOf(text: string): string {
  try {
    parseCycleImport(text, kinds)
    return ''
  } catch (e) {
    return importErrorMessage(e)
  }
}

function changed(change: (d: Record<string, any>) => void): string {
  const d = JSON.parse(cycleFile)
  change(d)
  return JSON.stringify(d)
}

describe('parseAnyImport', () => {
  it('tells the formats apart', () => {
    expect(parseAnyImport(cycleFile, kinds).kind).toBe('cycle')
    expect(parseAnyImport(trainingWeek, kinds).kind).toBe('training')
  })

  it('names both formats for anything else', () => {
    expect(() => parseAnyImport('{"format":"plan/2"}', kinds)).toThrow(
      'Expected "format": "training-import/1" (one training) or "cycle-import/1" (a whole cycle).',
    )
  })
})

describe('parseCycleImport', () => {
  it('reads a cycle with parallel trainings', () => {
    const c = parseCycleImport(cycleFile, kinds)
    expect(c.name).toBe('Mobility')
    expect(c.goals).toHaveLength(2)
    expect(c.trainings?.map((t) => [t.week, t.weeks])).toEqual([
      [1, 4],
      [1, 2],
      [3, 2],
    ])
  })

  it('accepts a cycle without trainings or goals', () => {
    expect(errorOf(changed((d) => (delete d.trainings, delete d.goals)))).toBe('')
  })

  it.each([
    ['a start inside a training', changed((d) => (d.trainings[2].start_date = '2026-11-16')), "Training 3: set the start on the cycle (start_date) and the training's week (week)."],
    ['a format inside a training', changed((d) => (d.trainings[0].format = 'training-import/1')), "Training 1: set the start on the cycle (start_date) and the training's week (week)."],
    ['week 0', changed((d) => (d.trainings[1].week = 0)), 'Training 2 ("Elbows 1"): week must be 1-4 (the cycle\'s weeks).'],
    ['a week past the cycle', changed((d) => (d.trainings[1].week = 5)), 'Training 2 ("Elbows 1"): week must be 1-4 (the cycle\'s weeks).'],
    ['weeks running past the cycle', changed((d) => (d.trainings[2].weeks = 3)), 'Training 3 ("Elbows 2"): 3 weeks from week 3 run past the cycle\'s 4 weeks.'],
    ['a training without a name', changed((d) => (d.trainings[0].name = '')), 'Training 1: needs a name.'],
    ['a broken session', changed((d) => (d.trainings[2].sessions[0].blocks[0].kind = 'foo')), 'Training 3 ("Elbows 2"), Elbows, block A: unknown kind "foo" (single, superset, complex, wod).'],
    ['a Tuesday start', changed((d) => (d.start_date = '2026-11-03')), 'start_date must be a Monday as YYYY-MM-DD.'],
    ['53 weeks', changed((d) => (d.weeks = 53)), 'weeks must be 1-52.'],
    ['a goal without a title', changed((d) => (d.goals[1].title = ' ')), 'Goal 2: needs a title (max 200 characters).'],
    ['an unknown goal status', changed((d) => (d.goals[0].status = 'done')), 'Goal 1: status must be open, achieved or missed.'],
    ['21 goals', changed((d) => (d.goals = Array(21).fill({ title: 'Goal' }))), 'A cycle has at most 20 goals.'],
  ])('rejects %s', (_, text, message) => {
    expect(errorOf(text)).toBe(message)
  })
})

describe('the cycle import request', () => {
  it('starts each training on its week of the cycle', () => {
    expect(trainingStart('2026-11-02', 1)).toBe('2026-11-02')
    expect(trainingStart('2026-11-02', 3)).toBe('2026-11-16')
  })

  it('merges the suggestions for new exercises, the first one winning', () => {
    const merged = mergedNewExercises(parseCycleImport(cycleFile, kinds))
    expect(merged.map((n) => n.name)).toEqual(['cossack squat ', 'Pigeon Pose'])
    expect(merged[0].type).toBe('mobility')
  })

  it('finds the unknown names across all trainings', () => {
    expect(unresolvedNames(parseCycleImport(cycleFile, kinds).trainings!, []).exercises).toEqual(['Cossack Squat', 'Pigeon Pose'])
  })

  it('builds the cycle and its trainings', () => {
    const { input, problem } = toCycleImportInput(parseCycleImport(cycleFile, kinds), byName('Cossack Squat', 'Pigeon Pose'))
    expect(problem).toBe('')
    expect(input!.cycle).toEqual({
      name: 'Mobility',
      description: 'Hips and elbows, side by side.',
      start_date: '2026-11-02',
      weeks: 4,
      goals: [
        { title: 'Better hips', target: 'Deep squat hold 2 min', status: 'open', notes: '' },
        { title: 'Better elbow extension', target: '', status: 'open', notes: 'Measure on video.' },
      ],
    })
    const t = input!.trainings
    expect(t.map((x) => [x.name, x.start_date, x.weeks, x.active, x.cycle_id])).toEqual([
      ['Hips', '2026-11-02', 4, true, null],
      ['Elbows 1', '2026-11-02', 2, false, null],
      ['Elbows 2', '2026-11-16', 2, true, null],
    ])
    // Each training plans its sessions in every one of its weeks.
    expect(t[0].schedule).toEqual(Array(4).fill([{ session: 0 }]))
    expect(t[0].sessions[0].blocks[0].exercises[0]).toMatchObject({ exercise_id: 'ex-Cossack Squat', sets: 3, reps_value: 8, per_side: true })
  })

  it('names the training a problem is in', () => {
    const c = parseCycleImport(changed((d) => delete d.trainings[2].sessions[0].blocks[0].exercises[0].reps), kinds)
    const { input, problem } = toCycleImportInput(c, byName('Cossack Squat', 'Pigeon Pose'))
    expect(input).toBeNull()
    expect(problem).toBe('Week 3 "Elbows 2": Elbows, block A: Pigeon Pose: Reps are required.')
  })
})

// The real program, when its files are in the repository's (git-ignored) import folder.
const local = (name: string) => fileURLToPath(new URL(`../../../../../import/${name}`, import.meta.url))
const masterDiesel = local('master-diesel-cycle.json')
const weekIntoCycle = local('master-diesel-week-01-into-cycle.json')

describe.skipIf(!existsSync(masterDiesel))('Master Diesel (local files)', () => {
  it('parses as a 10-week cycle with weekly trainings', () => {
    const c = parseCycleImport(readFileSync(masterDiesel, 'utf8'), kinds)
    expect([c.name, c.start_date, c.weeks, c.goals?.length, c.trainings?.length]).toEqual(['Master Diesel', '2026-10-26', 10, 3, 10])
    expect(c.trainings!.map((t) => trainingStart(c.start_date, t.week))).toEqual(
      Array.from({ length: 10 }, (_, i) => trainingStart('2026-10-26', i + 1)),
    )
    // Every exercise it names has a suggestion, so a fresh account can import it.
    const suggested = new Set(mergedNewExercises(c).map((n) => nameKey(n.name)))
    expect(unresolvedNames(c.trainings!, []).exercises.filter((n) => !suggested.has(nameKey(n)))).toEqual([])
  })

  it.skipIf(!existsSync(weekIntoCycle))('reads week 1 alone with its cycle', () => {
    const imp = parseAnyImport(readFileSync(weekIntoCycle, 'utf8'), kinds)
    expect(imp.kind).toBe('training')
    expect(imp.kind === 'training' && imp.imp.cycle).toBe('Master Diesel')
  })
})
