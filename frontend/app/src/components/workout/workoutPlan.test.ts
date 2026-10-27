import { describe, expect, it } from 'vitest'
import type { BlockKind, TrainingSession, Unit, WorkoutSet } from '../../types'
import { buildCards, progressOf, roundsOf, roundState, suggestedLoad, type LoadContext } from './workoutPlan'

const kg: Unit = { id: 'kg', name: 'kg', min_value: 0, max_value: null, kg_per_unit: 1, percent_of_max: false }
const lb: Unit = { id: 'lb', name: 'lb', min_value: 0, max_value: null, kg_per_unit: 0.45359237, percent_of_max: false }
const pct: Unit = { id: 'pct', name: '% 1RM', min_value: 0, max_value: 200, kg_per_unit: null, percent_of_max: true }
const units = new Map([kg, lb, pct].map((u) => [u.id, u]))
const single: BlockKind = {
  code: 'single',
  name: 'Single',
  description: '',
  min_exercises: 1,
  max_exercises: null,
  shared_sets: false,
  shared_reps: false,
  shared_load: false,
  scored: false,
}

// A session with one squat: 3 × 5 at 70–75 % 1RM.
const session: TrainingSession = {
  id: 's',
  name: 'Session 1',
  notes: '',
  blocks: [
    {
      id: 'b',
      kind: 'single',
      name: '',
      sets: null,
      notes: '',
      reps_value: null,
      reps_max: null,
      reps_unit: null,
      per_side: false,
      load_value: null,
      load_max: null,
      load_unit: null,
      load_bodyweight: false,
      reps_unit_name: '',
      load_unit_name: '',
      score_type: null,
      time_seconds: null,
      interval_seconds: null,
      exercises: [
        {
          exercise_id: 'squat',
          name: 'Back squat',
          type: 'strength',
          sets: 3,
          tempo: '',
          notes: '',
          reps_value: 5,
          reps_max: null,
          reps_unit: 'reps',
          per_side: false,
          load_value: 70,
          load_max: 75,
          load_unit: 'pct',
          load_bodyweight: false,
          reps_unit_name: 'reps',
          load_unit_name: '% 1RM',
        },
      ],
    },
  ],
}

const context = (max: number | null, unit = kg): LoadContext => ({
  unitOf: (id) => units.get(id),
  maxOf: () => (max === null ? null : { value: max, unit }),
})
const cardOf = (sets: WorkoutSet[], loads?: LoadContext) =>
  buildCards(session, sets, [], () => single, 'kg', loads).blocks[0].exercises[0]

describe('buildCards with % 1RM', () => {
  it('calculates the load from the 1RM, in its unit', () => {
    const card = cardOf([], context(120))
    expect(card.loadUnit).toBe('kg')
    expect(card.rows.map((r) => r.load)).toEqual([84, 84, 84])
    expect(card.percentPlan).toEqual({ value: 70, max: 75 })

    const inLb = cardOf([], context(300, lb))
    expect(inLb.loadUnit).toBe('lb')
    expect(inLb.rows[0].load).toBe(210)
  })

  it('keeps the plan in % 1RM without a max', () => {
    const card = cardOf([], context(null))
    expect(card.loadUnit).toBe('pct')
    expect(card.rows[0].load).toBe(70)
  })

  it('converts the plan to the unit sets were logged in', () => {
    const logged: WorkoutSet = {
      id: 'set1',
      exercise_id: 'squat',
      block_position: 1,
      exercise_position: 1,
      set_number: 1,
      reps_value: 5,
      reps_unit: 'reps',
      per_side: false,
      load_value: 185,
      load_unit: 'lb',
      bodyweight: false,
      completed_at: '',
      exercise_name: 'Back squat',
      reps_unit_name: 'reps',
      load_unit_name: 'lb',
    }
    const card = cardOf([logged], context(120))
    expect(card.loadUnit).toBe('lb')
    expect(card.rows.map((r) => r.load)).toEqual([185, 185, 185]) // 84 kg = 185.2 lb
  })
})

// The squat planned without a load: 3 × 5.
const unloadedSession: TrainingSession = {
  ...session,
  blocks: [{ ...session.blocks[0], exercises: [{ ...session.blocks[0].exercises[0], load_value: null, load_max: null, load_unit: null }] }],
}
const withLast = (value: number, unit: Unit): LoadContext => ({ ...context(null), lastOf: () => ({ value, unit }) })
const unitOf = (id: string) => units.get(id)

describe('the load logged last time', () => {
  it('is suggested in its unit when the plan has no load', () => {
    const card = buildCards(unloadedSession, [], [], () => single, 'kg', withLast(135, lb)).blocks[0].exercises[0]
    expect(card.loadUnit).toBe('lb')
    expect(card.rows.map((r) => r.load)).toEqual([null, null, null]) // only a placeholder
    expect(suggestedLoad(card, unitOf, null)).toBe(135)
    expect(suggestedLoad({ ...card, loadUnit: 'kg' }, unitOf, null)).toBe(61) // 61.2 kg
  })

  it('is not suggested on bodyweight', () => {
    const plan = structuredClone(unloadedSession)
    plan.blocks[0].exercises[0].load_bodyweight = true
    const card = buildCards(plan, [], [], () => single, 'pct', { ...withLast(20, kg), weightUnit: 'kg' }).blocks[0].exercises[0]
    expect(card.lastLoad).toBeNull()
  })

  it('is not suggested when the plan has a load', () => {
    const card = buildCards(session, [], [], () => single, 'kg', withLast(100, kg)).blocks[0].exercises[0]
    expect(card.lastLoad).toBeNull()
    expect(suggestedLoad(card, unitOf, null)).toBeNull()
  })
})

// A complex: 3 rounds of a power clean and a front squat at 60 kg, sharing sets, reps and load.
const complex: BlockKind = { ...single, code: 'complex', name: 'Complex', shared_sets: true, shared_reps: true, shared_load: true }
const plannedExercise = session.blocks[0].exercises[0]
const complexSession: TrainingSession = {
  ...session,
  blocks: [
    {
      ...session.blocks[0],
      kind: 'complex',
      sets: 3,
      reps_value: 1,
      reps_unit: 'reps',
      load_value: 60,
      load_unit: 'kg',
      exercises: ['clean', 'squat'].map((id) => ({
        ...plannedExercise,
        exercise_id: id,
        sets: null,
        reps_value: null,
        reps_unit: null,
        load_value: null,
        load_max: null,
        load_unit: null,
      })),
    },
  ],
}
const cleanSet: WorkoutSet = {
  id: 'set1',
  exercise_id: 'clean',
  block_position: 1,
  exercise_position: 1,
  set_number: 2,
  reps_value: 1,
  reps_unit: 'reps',
  per_side: false,
  load_value: 65,
  load_unit: 'kg',
  bodyweight: false,
  completed_at: '',
  exercise_name: 'Power clean',
  reps_unit_name: 'reps',
  load_unit_name: 'kg',
}
const complexBlock = (sets: WorkoutSet[] = []) =>
  buildCards(complexSession, sets, [], () => complex, 'kg', context(null)).blocks[0]

describe('a complex', () => {
  it('is logged in rounds of all its exercises', () => {
    const block = complexBlock()
    expect(block.complex).toBe(true)
    const rounds = roundsOf(block)
    expect(rounds.map((r) => r.number)).toEqual([1, 2, 3])
    expect(rounds[0].sets.map((s) => s.card.exerciseId)).toEqual(['clean', 'squat'])
    expect(progressOf(block)).toEqual({ logged: 0, total: 3 })
  })

  it('counts a round as logged only when all its sets are', () => {
    const block = complexBlock([cleanSet])
    const round = roundsOf(block)[1]
    expect(roundState(round)).toBe('edited')
    expect(roundState(roundsOf(block)[0])).toBe('open')

    const squatSet = { ...cleanSet, id: 'set2', exercise_id: 'squat', exercise_position: 2, exercise_name: 'Front squat' }
    const done = complexBlock([cleanSet, squatSet])
    expect(roundState(roundsOf(done)[1])).toBe('logged')
    expect(progressOf(done)).toEqual({ logged: 1, total: 3 })
    roundsOf(done)[1].sets[1].row.load = 70
    expect(roundState(roundsOf(done)[1])).toBe('edited')
  })

  it('starts a set not logged yet with the load the other exercise was logged with', () => {
    const [, squat] = complexBlock([cleanSet]).exercises
    expect(squat.rows.map((r) => r.load)).toEqual([60, 65, 60])
  })

  it('is only a kind that shares the load', () => {
    const block = buildCards(complexSession, [], [], () => single, 'kg', context(null)).blocks[0]
    expect(block.complex).toBe(false)
    expect(progressOf(block)).toEqual({ logged: 0, total: 6 }) // every exercise's own sets
  })
})

describe('an exercise on bodyweight', () => {
  const onBodyweight = (load: { load_value: number | null; load_unit: string | null; load_unit_name: string }) => {
    const plan = structuredClone(unloadedSession)
    Object.assign(plan.blocks[0].exercises[0], { load_bodyweight: true, load_max: null, ...load })
    return plan
  }
  const context = { ...withLast(20, kg), weightUnit: 'kg' }

  it('logs only the reps without weight added', () => {
    const card = buildCards(onBodyweight({ load_value: null, load_unit: null, load_unit_name: '' }), [], [], () => single, 'pct', context)
      .blocks[0].exercises[0]
    expect(card.bodyweight).toBe(true)
    expect(card.showLoad).toBe(false)
    expect(card.planned).toBe('3 × 5 reps · bodyweight')
    expect(card.loadUnit).toBe('kg') // where weight added in the workout starts, not % 1RM
  })

  it('shows the weight planned on top', () => {
    const card = buildCards(onBodyweight({ load_value: 10, load_unit: 'kg', load_unit_name: 'kg' }), [], [], () => single, 'kg', context)
      .blocks[0].exercises[0]
    expect(card.showLoad).toBe(true)
    expect(card.planned).toBe('3 × 5 reps · bodyweight + 10 kg')
    expect(card.rows.map((r) => r.load)).toEqual([10, 10, 10])
  })

  it('shows the weight logged on top', () => {
    const logged: WorkoutSet = { ...cleanSet, exercise_id: 'squat', set_number: 1, reps_value: 5, load_value: 5, bodyweight: true }
    const card = buildCards(onBodyweight({ load_value: null, load_unit: null, load_unit_name: '' }), [logged], [], () => single, 'kg', context)
      .blocks[0].exercises[0]
    expect(card.bodyweight).toBe(true)
    expect(card.showLoad).toBe(true)
    expect(card.rows.map((r) => r.load)).toEqual([5, null, null])
  })
})
