import { describe, expect, it } from 'vitest'
import type { ExerciseMax, Unit } from '../../types'
import { convertLoad, isNewMax, newestFirst, prIds, roundLoad } from './maxes'

const kg: Unit = { id: 'kg', name: 'kg', min_value: 0, max_value: null, kg_per_unit: 1, percent_of_max: false }
const lb: Unit = { id: 'lb', name: 'lb', min_value: 0, max_value: null, kg_per_unit: 0.45359237, percent_of_max: false }
const pct: Unit = { id: 'pct', name: '% 1RM', min_value: 0, max_value: 200, kg_per_unit: null, percent_of_max: true }
const rpe: Unit = { id: 'rpe', name: 'RPE', min_value: 1, max_value: 10, kg_per_unit: null, percent_of_max: false }
const units = new Map([kg, lb, pct, rpe].map((u) => [u.id, u]))

const max = (id: string, value: number, unit: string, day: string, created = '2026-01-01T00:00:00Z'): ExerciseMax => ({
  id,
  exercise_id: 'squat',
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

describe('isNewMax', () => {
  const current = { value: 100, unit: kg }
  it('is a single above the current max', () => {
    expect(isNewMax(1, 102.5, kg, current)).toBe(true)
    expect(isNewMax(1, 225, lb, current)).toBe(true) // 102.06 kg
    expect(isNewMax(1, 100, kg, current)).toBe(false)
    expect(isNewMax(2, 110, kg, current)).toBe(false)
    expect(isNewMax(1, 90, pct, current)).toBe(false)
  })

  it('is any weighted single without a max yet', () => {
    expect(isNewMax(1, 60, kg, null)).toBe(true)
    expect(isNewMax(1, null, kg, null)).toBe(false)
  })
})
