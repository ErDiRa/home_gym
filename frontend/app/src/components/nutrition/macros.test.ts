import { describe, expect, it } from 'vitest'
import type { DayMacros, MacroTarget } from '../../types'
import { kcalOf, macroRows, macrosLine, shareTone } from './macros'

const day = (m: Partial<DayMacros>): DayMacros => ({
  date: '2026-11-02',
  protein_g: null,
  fat_g: null,
  carbs_g: null,
  kcal: 0,
  notes: '',
  updated_at: '2026-11-02T20:00:00Z',
  ...m,
})

const target: MacroTarget = { kcal: 2600, protein_g: 160, fat_g: 72, carbs_g: 329, source: 'estimate', weight_kg: 80 }

describe('macros', () => {
  it('computes the energy like the server', () => {
    expect(kcalOf({ protein_g: 160, fat_g: 70, carbs_g: null })).toBe(160 * 4 + 70 * 9)
    expect(kcalOf({ protein_g: null, fat_g: null, carbs_g: null })).toBe(0)
  })

  it('compares what was eaten with the target', () => {
    const rows = macroRows(day({ protein_g: 120, carbs_g: 329 }), target)
    expect(rows.map((r) => r.share)).toEqual([0.75, null, 1])
    expect(rows.map((r) => shareTone(r.share))).toEqual(['low', 'none', 'ok'])
    expect(macroRows(null, null).every((r) => r.eaten === null && r.target === null)).toBe(true)
    expect(shareTone(1.2)).toBe('high')
  })

  it('writes a compact line of the macros logged', () => {
    expect(macrosLine(day({ protein_g: 160.4, carbs_g: 1250 }))).toBe('P 160 · C 1,250 g')
  })
})
