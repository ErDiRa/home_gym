import { describe, expect, it } from 'vitest'
import { addMonths, formatMonth, isMonth, monthGrid } from './dates'

describe('months', () => {
  it('steps across years', () => {
    expect(addMonths('2026-12', 1)).toBe('2027-01')
    expect(addMonths('2026-01', -1)).toBe('2025-12')
    expect(addMonths('2026-10', 0)).toBe('2026-10')
  })

  it('checks and formats YYYY-MM', () => {
    expect(isMonth('2026-10')).toBe(true)
    expect(isMonth('2026-13')).toBe(false)
    expect(isMonth('2026-10-01')).toBe(false)
    expect(formatMonth('2026-10')).toBe('October 2026')
  })
})

describe('monthGrid', () => {
  it('fills whole weeks from Monday to Sunday', () => {
    // October 2026 starts on a Thursday and ends on a Saturday; it holds the DST change in Europe.
    const days = monthGrid('2026-10')
    expect(days[0]).toBe('2026-09-28')
    expect(days[days.length - 1]).toBe('2026-11-01')
    expect(days).toHaveLength(35)
    expect(new Set(days).size).toBe(35)
  })

  it('starts on the first when the month starts on a Monday', () => {
    const days = monthGrid('2027-02') // 1 Feb 2027 is a Monday, 28 Feb a Sunday
    expect(days[0]).toBe('2027-02-01')
    expect(days).toHaveLength(28)
  })

  it('takes six weeks when a long month starts on a Sunday', () => {
    const days = monthGrid('2026-03') // 1 Mar 2026 is a Sunday
    expect(days[0]).toBe('2026-02-23')
    expect(days[days.length - 1]).toBe('2026-04-05')
    expect(days).toHaveLength(42)
  })
})
