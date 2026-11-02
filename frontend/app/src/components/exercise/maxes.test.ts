import { describe, expect, it } from 'vitest'
import type { ExerciseMax, Unit } from '../../types'
import { convertLoad, current1Rm, effectiveMax, estimateOneRm, newestFirst, newMaxOffers, oneRmOf, prIds, repMaxes, roundLoad } from './maxes'

const kg: Unit = { id: 'kg', name: 'kg', min_value: 0, max_value: null, kg_per_unit: 1, percent_of_max: false }
const lb: Unit = { id: 'lb', name: 'lb', min_value: 0, max_value: null, kg_per_unit: 0.45359237, percent_of_max: false }
const pct: Unit = { id: 'pct', name: '% 1RM', min_value: 0, max_value: 200, kg_per_unit: null, percent_of_max: true }
const rpe: Unit = { id: 'rpe', name: 'RPE', min_value: 1, max_value: 10, kg_per_unit: null, percent_of_max: false }
const units = new Map([kg, lb, pct, rpe].map((u) => [u.id, u]))

const max = (id: string, value: number, unit: string, day: string, created = '2026-01-01T00:00:00Z', reps = 1): ExerciseMax => ({
  id,
  exercise_id: 'squat',
  reps,
  load_value: value,
  load_unit: unit,
  load_unit_name: unit,
  achieved_on: day,
  workout_set_id: null,
  notes: '',
  created_at: created,
})

describe('convertLoad', () => {
  it('converts between kg and lb', () => {
    expect(convertLoad(100, kg, lb, null)).toBeCloseTo(220.462, 3)
    expect(convertLoad(225, lb, kg, null)).toBeCloseTo(102.058, 3)
    expect(roundLoad(convertLoad(100, kg, lb, null)!)).toBe(220.5)
  })

  it('turns % 1RM into a weight with the max, in any weight unit', () => {
    expect(convertLoad(70, pct, kg, { value: 120, unit: kg })).toBeCloseTo(84)
    expect(convertLoad(70, pct, lb, { value: 120, unit: kg })).toBeCloseTo(185.188, 3)
    // A max entered in lb works the same.
    expect(convertLoad(50, pct, kg, { value: 300, unit: lb })).toBeCloseTo(68.039, 3)
    expect(convertLoad(84, kg, pct, { value: 120, unit: kg })).toBeCloseTo(70)
  })

  it('gives up without a max or for units that are no weight', () => {
    expect(convertLoad(70, pct, kg, null)).toBeNull()
    expect(convertLoad(8, rpe, kg, { value: 120, unit: kg })).toBeNull()
    expect(convertLoad(100, kg, rpe, null)).toBeNull()
    expect(convertLoad(8, rpe, rpe, null)).toBe(8)
  })
})

describe('history', () => {
  const history = newestFirst([
    max('a', 100, 'kg', '2026-01-10'),
    max('b', 95, 'kg', '2026-03-01'), // after a break: current, but no PR
    max('c', 230, 'lb', '2026-06-01'), // 104.3 kg: a PR
    max('d', 105, 'kg', '2026-06-01', '2026-06-01T12:00:00Z'), // same day, entered later
  ])

  it('sorts newest first, the current max first', () => {
    expect(history.map((m) => m.id)).toEqual(['d', 'c', 'b', 'a'])
  })

  it('marks the entries that beat every earlier one, across units', () => {
    expect([...prIds(history, (id) => units.get(id))].sort()).toEqual(['a', 'c', 'd'])
  })
})

const unitOf = (id: string) => units.get(id)

describe('rep maxes', () => {
  // A 1RM of 140 kg, 5RMs of 110 and later 115 kg, and a 3RM of 125 kg.
  const history = newestFirst([
    max('one', 140, 'kg', '2026-01-10'),
    max('five-a', 110, 'kg', '2026-02-01', '2026-02-01T00:00:00Z', 5),
    max('three', 125, 'kg', '2026-03-01', '2026-03-01T00:00:00Z', 3),
    max('five-b', 115, 'kg', '2026-04-01', '2026-04-01T00:00:00Z', 5),
  ])

  it('estimates a 1RM with Epley: weight x (1 + reps / 30)', () => {
    expect(estimateOneRm(100, 1)).toBe(100)
    expect(estimateOneRm(100, 3)).toBeCloseTo(110)
    expect(estimateOneRm(100, 5)).toBeCloseTo(116.667, 3)
    expect(oneRmOf(max('x', 100, 'kg', '2026-01-01', undefined, 5))).toBe(116.5) // rounded to half a kg
    expect(oneRmOf(max('x', 100, 'kg', '2026-01-01'))).toBe(100)
  })

  it('compares a PR only with the entries of the same reps', () => {
    // The 5RM of 115 beats the one of 110; the 3RM and the 1RM are firsts of their own.
    expect([...prIds(history, unitOf)].sort()).toEqual(['five-a', 'five-b', 'one', 'three'])
    expect([...prIds(newestFirst([max('a', 120, 'kg', '2026-01-01', undefined, 5), max('b', 100, 'kg', '2026-02-01')]), unitOf)].sort()).toEqual(['a', 'b'])
  })

  it('lists the best entry of each rep count, by reps', () => {
    expect(repMaxes(history, unitOf).map((m) => m.id)).toEqual(['one', 'three', 'five-b'])
  })

  it('takes the 1RM from the newest entry of one rep only', () => {
    expect(current1Rm(history)?.id).toBe('one')
    expect(current1Rm(history.filter((m) => m.reps !== 1))).toBeNull()
  })

  it('calculates with the real 1RM, else the best estimate of a rep max', () => {
    expect(effectiveMax(history, unitOf)).toMatchObject({ load_value: 140, estimated: false, reps: 1 })
    // 3RM 125 kg -> 137.5; 5RM 115 kg -> 134.2: the 3RM estimates higher.
    expect(effectiveMax(history.filter((m) => m.reps !== 1), unitOf)).toMatchObject({ load_value: 137.5, load_unit: 'kg', estimated: true, reps: 3 })
    expect(effectiveMax([], unitOf)).toBeNull()
  })

  it('compares estimates across units', () => {
    // 5RM 250 lb = 113.4 kg -> 132.3 kg; 3RM 120 kg -> 132 kg: the 5RM in lb wins, shown in lb.
    const mixed = [max('lb5', 250, 'lb', '2026-01-01', undefined, 5), max('kg3', 120, 'kg', '2026-01-02', undefined, 3)]
    expect(effectiveMax(mixed, unitOf)).toMatchObject({ load_value: 291.5, load_unit: 'lb', estimated: true, reps: 5 })
  })
})

describe('newMaxOffers', () => {
  const set = (reps: number, load: number, unit = 'kg') => ({ reps_value: reps, load_value: load, load_unit: unit })
  const current = [max('one', 100, 'kg', '2026-01-01')]

  it('offers the heaviest single, not the first or the last logged', () => {
    expect(newMaxOffers([set(1, 105), set(1, 110), set(1, 102.5)], unitOf, current)).toEqual([{ set: set(1, 110), reps: 1 }])
    expect(newMaxOffers([set(1, 102.5), set(1, 105)], unitOf, current)).toEqual([{ set: set(1, 105), reps: 1 }])
  })

  it('offers a single and a 5-rep set together, by reps', () => {
    const offers = newMaxOffers([set(5, 90), set(1, 110), set(5, 85)], unitOf, current)
    expect(offers).toEqual([
      { set: set(1, 110), reps: 1 },
      { set: set(5, 90), reps: 5 },
    ])
  })

  it('beats the best entry of the same reps, in kg across units', () => {
    const entries = [max('one', 100, 'kg', '2026-01-01'), max('five', 90, 'kg', '2026-01-02', undefined, 5)]
    // 195 lb = 88.5 kg: no new 5RM; 205 lb = 93 kg is; a 3 of 80 kg is the first 3RM.
    expect(newMaxOffers([set(5, 195, 'lb')], unitOf, entries)).toEqual([])
    expect(newMaxOffers([set(5, 205, 'lb'), set(3, 80)], unitOf, entries)).toEqual([
      { set: set(3, 80), reps: 3 },
      { set: set(5, 205, 'lb'), reps: 5 },
    ])
    expect(newMaxOffers([set(1, 100)], unitOf, entries)).toEqual([]) // equal is no new max
  })

  it('offers the first max when there is none, only for usual rep counts and weights', () => {
    expect(newMaxOffers([set(1, 60), set(1, 70)], unitOf, [])).toEqual([{ set: set(1, 70), reps: 1 }])
    expect(newMaxOffers([set(4, 100), set(8, 80), set(1, 90, 'rpe')], unitOf, [])).toEqual([])
    expect(newMaxOffers([{ reps_value: 1, load_value: null, load_unit: null }], unitOf, [])).toEqual([])
  })
})
