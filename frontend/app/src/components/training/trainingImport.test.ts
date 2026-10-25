import { beforeEach, describe, expect, it } from 'vitest'
import trainingWeek from './__fixtures__/training-week.json?raw'
import intoCycle from './__fixtures__/training-week-into-cycle.json?raw'
import { byName, exercise, useTestLookups } from './__fixtures__/lookups'
import { sessionProblem } from './trainingForm'
import { importErrorMessage, parseImport, toImportedTraining, unresolvedNames } from './trainingImport'

let kinds: string[]
beforeEach(() => {
  kinds = useTestLookups()
})

// The error message parseImport throws for the text, or '' when it parses.
function errorOf(text: string): string {
  try {
    parseImport(text, kinds)
    return ''
  } catch (e) {
    return importErrorMessage(e)
  }
}

// The fixture with a change made to its parsed JSON.
function changed(change: (d: Record<string, any>) => void): string {
  const d = JSON.parse(trainingWeek)
  change(d)
  return JSON.stringify(d)
}

describe('parseImport', () => {
  it('reads a training week', () => {
    const imp = parseImport(trainingWeek, kinds)
    expect(imp.name).toBe('Strength week')
    expect(imp.sessions).toHaveLength(2)
    expect(imp.cycle).toBeUndefined()
  })

  it('reads the name of the cycle it goes into', () => {
    expect(parseImport(intoCycle, kinds).cycle).toBe('  Mobility ')
  })

  it.each([
    ['not JSON', '{', 'This is not valid JSON.'],
    ['not an object', '[]', 'Expected a JSON object.'],
    ['another format', changed((d) => (d.format = 'cycle-import/1')), 'Expected "format": "training-import/1".'],
    ['an empty cycle', changed((d) => (d.cycle = ' ')), 'cycle must be the name of a cycle.'],
    ['no name', changed((d) => delete d.name), 'The training needs a name.'],
    ['a Tuesday', changed((d) => (d.start_date = '2026-11-03')), 'start_date must be a Monday as YYYY-MM-DD.'],
    ['no sessions', changed((d) => (d.sessions = [])), 'There are no sessions.'],
    ['8 sessions', changed((d) => (d.sessions = Array(8).fill(d.sessions[0]))), 'A week holds at most 7 sessions.'],
    [
      'an unknown kind',
      changed((d) => (d.sessions[1].blocks[0].kind = 'foo')),
      'Thu, block A: unknown kind "foo" (single, superset, complex, wod).',
    ],
    [
      'an unknown score type',
      changed((d) => (d.sessions[0].blocks[1].score.type = 'tabata')),
      'Mon, block B: unknown score type "tabata" (for_time, amrap, max_reps, max_calories, emom).',
    ],
    ['an amount without unit', changed((d) => delete d.sessions[0].blocks[0].exercises[0].reps.unit), 'Mon, block A, Squat, reps: unit is missing.'],
  ])('rejects %s', (_, text, message) => {
    expect(errorOf(text)).toBe(message)
  })
})

describe('unresolvedNames', () => {
  it('lists unknown exercises and units once', () => {
    const imp = parseImport(trainingWeek, kinds)
    const known = [exercise('Squat'), exercise('burpee')]
    expect(unresolvedNames(imp, known)).toEqual({ exercises: ['Wall Ball', 'Ring Row', 'Copenhagen Plank'], units: [] })
  })

  it('covers several trainings at once', () => {
    const a = parseImport(trainingWeek, kinds)
    const b = parseImport(changed((d) => (d.sessions[0].blocks[0].exercises[0].exercise = 'Deadlift')), kinds)
    expect(unresolvedNames([a, b], [exercise('Squat'), exercise('Burpee'), exercise('Wall Ball'), exercise('Ring Row')]).exercises).toEqual([
      'Copenhagen Plank',
      'Deadlift',
    ])
  })

  it('lists unknown units by part', () => {
    const imp = parseImport(changed((d) => (d.sessions[1].blocks[0].exercises[0].reps.unit = 'ft')), kinds)
    expect(unresolvedNames(imp, []).units).toEqual([{ part: 'reps', name: 'ft' }])
  })
})

describe('toImportedTraining', () => {
  const names = byName('Squat', 'Wall Ball', 'Burpee', 'Ring Row', 'Copenhagen Plank')

  it('builds the form rows, planned once in the week', () => {
    const t = toImportedTraining(parseImport(trainingWeek, kinds), names)
    expect(t.name).toBe('Strength week')
    expect(t.startDate).toBe('2026-11-02')
    expect(t.sessions.map((s) => s.name)).toEqual(['Mon', 'Thu'])
    expect(t.weeks).toHaveLength(1)
    expect(t.weeks[0].slots.map((s) => s.sessionKey)).toEqual(t.sessions.map((s) => s.key))
    const squat = t.sessions[0].blocks[0].exercises[0]
    expect(squat).toMatchObject({ exercise_id: 'ex-Squat', sets: 5, reps_value: 3, reps_unit: 'unit-reps', load_value: 75, load_max: 80, tempo: '31X1' })
    // The WOD keeps its rounds and score; the form's own checks find nothing.
    expect(t.sessions[0].blocks[1]).toMatchObject({ rounds: 3, score_type: 'for_time', time_seconds: 600 })
    expect(t.sessions.map(sessionProblem)).toEqual(['', ''])
  })

  it('keeps the cycle named in the file, trimmed', () => {
    expect(toImportedTraining(parseImport(intoCycle, kinds), names).cycle).toBe('Mobility')
  })

  it('plans the sessions in every week', () => {
    const t = toImportedTraining(parseImport(trainingWeek, kinds), names, 3)
    expect(t.weeks).toHaveLength(3)
    expect(t.weeks.every((w) => w.slots.length === 2)).toBe(true)
  })
})
