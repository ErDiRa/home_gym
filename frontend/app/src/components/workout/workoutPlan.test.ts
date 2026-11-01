import { describe, expect, it } from 'vitest'
import type { BlockKind, TrainingSession, Unit, WorkoutSet } from '../../types'
import { buildCards, changed, listMark, progressTotal, scoreDraft, scoreValues, takesProgress, progressOf, roundsOf, roundState, suggestedLoad, type LoadContext } from './workoutPlan'

const kg: Unit = { id: 'kg', name: 'kg', min_value: 0, max_value: null, kg_per_unit: 1, percent_of_max: false }
const lb: Unit = { id: 'lb', name: 'lb', min_value: 0, max_value: null, kg_per_unit: 0.45359237, percent_of_max: false }
const pct: Unit = { id: 'pct', name: '% 1RM', min_value: 0, max_value: 200, kg_per_unit: null, percent_of_max: true }
const ft: Unit = { id: 'ft', name: 'ft', min_value: 0, max_value: null, kg_per_unit: null, percent_of_max: false }
const units = new Map([kg, lb, pct, ft].map((u) => [u.id, u]))
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
          intensity_metric: null,
          intensity_value: null,
          intensity_max: null,
          recovery_value: null,
          recovery_max: null,
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
      intensity_metric: null,
      intensity_value: null,
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
  intensity_metric: null,
  intensity_value: null,
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

describe('an endurance exercise', () => {
  // 4 × 4 min on the rower, logged in cal, at 1:55–2:00 /500 m with 2:30 recovery.
  const rowing = () => {
    const plan = structuredClone(unloadedSession)
    Object.assign(plan.blocks[0].exercises[0], {
      exercise_id: 'row',
      name: 'Row Erg',
      type: 'endurance',
      sets: 4,
      reps_value: 4,
      reps_unit_name: 'min',
      load_unit: 'cal',
      load_unit_name: 'cal',
      intensity_metric: 'pace_500m',
      intensity_value: 115,
      intensity_max: 120,
      recovery_value: 150,
    })
    return plan
  }

  it('shows the target and prefills the average with it', () => {
    const card = buildCards(rowing(), [], [], () => single, 'kg', context(null)).blocks[0].exercises[0]
    expect(card.planned).toBe('4 × 4 min · in cal · @ 1:55–2:00 /500 m · recovery 2:30 /500 m')
    expect(card.endurance).toBe(true)
    expect(card.intensityMetric).toBe('pace_500m')
    expect(card.rows.map((r) => r.intensity)).toEqual([115, 115, 115, 115])
  })

  it('keeps the metric logged in and counts a changed average as an edit', () => {
    const logged: WorkoutSet = {
      ...cleanSet,
      exercise_id: 'row',
      set_number: 1,
      reps_value: 4,
      load_value: 62,
      load_unit: 'cal',
      intensity_metric: 'watts',
      intensity_value: 231,
    }
    const card = buildCards(rowing(), [logged], [], () => single, 'kg', context(null)).blocks[0].exercises[0]
    expect(card.intensityMetric).toBe('watts')
    expect(card.rows.map((r) => r.intensity)).toEqual([231, null, null, null]) // the target is a pace, not watts
    const row = card.rows[0]
    expect(changed(card, row)).toBe(false)
    row.intensity = 240
    expect(changed(card, row)).toBe(true)
  })

  it('is no endurance exercise without the type', () => {
    expect(buildCards(unloadedSession, [], [], () => single, 'kg', context(null)).blocks[0].exercises[0].endurance).toBe(false)
  })
})

// A WOD alternating hang cleans (135 lb) and rope climbs, three rounds each.
const wodKind: BlockKind = { ...single, code: 'wod', name: 'WOD', shared_sets: true, scored: true }
const ladder = (id: string, name: string, reps: number, load: number | null, unit = 'lb'): TrainingSession['blocks'][number]['exercises'][number] => ({
  ...plannedExercise,
  exercise_id: id,
  name,
  sets: null,
  reps_value: reps,
  load_value: load,
  load_max: null,
  load_unit: load === null ? null : unit,
  load_unit_name: load === null ? '' : unit,
})
const wodSession: TrainingSession = {
  ...session,
  blocks: [
    {
      ...session.blocks[0],
      kind: 'wod',
      score_type: 'for_time',
      exercises: [ladder('clean', 'Hang Clean', 18, 135), ladder('rope', 'Rope Climb', 1, 15, 'ft'), ladder('clean', 'Hang Clean', 15, 135), ladder('rope', 'Rope Climb', 2, 15, 'ft')],
    },
  ],
}
const wodContext = { ...context(null), weightUnit: 'kg' }

describe('a WOD', () => {
  const wod = () => buildCards(wodSession, [], [], () => wodKind, 'kg', wodContext).blocks[0].wod!

  it('lists the exercises in plan order, the buttons on each one\'s first line', () => {
    const lines = wod().lines
    expect(lines.map((l) => `${l.reps} ${l.name}`)).toEqual(['18 Hang Clean', '1 Rope Climb', '15 Hang Clean', '2 Rope Climb'])
    expect(lines.map((l) => l.first)).toEqual([true, true, false, false])
    expect(lines[0].load).toBe('135 lb')
    expect(lines[1].load).toBe('15 ft')
  })

  describe('capped', () => {
    const capped = (reps: number | null) => ({
      id: 'r', workout_id: 'w', block_position: 1, score_type: 'for_time' as const, time_seconds: null, rounds: null, reps, calories: null, notes: '', completed_at: '',
    })
    // Lines of 18, 1, 15 and 2 reps: 36 reps in the round.
    const cappedWod = (reps: number | null) =>
      buildCards(wodSession, [], reps === null ? [] : [capped(reps)], () => wodKind, 'kg', wodContext).blocks[0].wod!

    it('takes how far it got, for lines of plain reps', () => {
      expect(takesProgress(cappedWod(null))).toBe(true)
    })

    it('shows a saved score as the progress that makes it, and sends the same reps', () => {
      const wod = cappedWod(20) // 18 + 1 done, 1 of the next 15
      expect(wod.draft.capped).toBe(true)
      expect(wod.draft.progress).toEqual({ rounds: 0, step: 2, reps: 1 })
      expect(progressTotal(wod)).toBe(20)
      expect(scoreValues('for_time', wod.draft)).toEqual({ time_seconds: null, rounds: null, reps: 20, calories: null })
      expect(listMark(wod)).toEqual({ step: 2, reps: 1 })
    })

    it('marks nothing without a capped score, and starts at the beginning', () => {
      const wod = cappedWod(null)
      expect(listMark(wod)).toBeNull()
      expect(wod.draft.progress).toEqual({ rounds: 0, step: 0, reps: 0 })
      expect(scoreDraft(null, wod.lines, wod.rounds).capped).toBe(false)
    })

    it('has no progress for a line that is not plain reps', () => {
      const withFeet = { ...wodSession, blocks: [{ ...wodSession.blocks[0], exercises: [ladder('clean', 'Hang Clean', 18, 135), { ...ladder('walk', 'Handstand Walk', 25, null), reps_unit_name: 'ft' }] }] }
      const wod = buildCards(withFeet, [], [], () => wodKind, 'kg', wodContext).blocks[0].wod!
      expect(takesProgress(wod)).toBe(false)
    })
  })
})
