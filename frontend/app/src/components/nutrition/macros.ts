import type { DayMacros, MacroTarget, MacrosInput } from '../../types'

export type MacroKey = 'protein_g' | 'fat_g' | 'carbs_g'

export const MACROS: { key: MacroKey; label: string; short: string; kcalPerGram: number }[] = [
  { key: 'protein_g', label: 'Protein', short: 'P', kcalPerGram: 4 },
  { key: 'fat_g', label: 'Fat', short: 'F', kcalPerGram: 9 },
  { key: 'carbs_g', label: 'Carbs', short: 'C', kcalPerGram: 4 },
]

// The energy of the macros entered (4/9/4 kcal per gram), as the server computes it.
export function kcalOf(m: Pick<MacrosInput, MacroKey>): number {
  return MACROS.reduce((sum, { key, kcalPerGram }) => sum + (m[key] ?? 0) * kcalPerGram, 0)
}

export interface MacroRow {
  key: MacroKey
  label: string
  short: string
  eaten: number | null
  target: number | null
  // eaten / target, from 0; null without both
  share: number | null
}

// One row per macro: what was eaten against the target.
export function macroRows(macros: DayMacros | null, target: MacroTarget | null): MacroRow[] {
  return MACROS.map(({ key, label, short }) => {
    const eaten = macros?.[key] ?? null
    const goal = target?.[key] ?? null
    return { key, label, short, eaten, target: goal, share: eaten !== null && goal ? eaten / goal : null }
  })
}

// A share of the target is on track within ±10 %.
export function shareTone(share: number | null): 'none' | 'low' | 'ok' | 'high' {
  if (share === null) return 'none'
  if (share < 0.9) return 'low'
  return share > 1.1 ? 'high' : 'ok'
}

// Grams or kcal, whole, with thousands separators: "1,850".
export const formatAmount = (n: number) => Math.round(n).toLocaleString('en-US')

// A compact line of the macros eaten, e.g. "P 160 · F 70 · C 250 g".
export function macrosLine(m: DayMacros): string {
  const parts = MACROS.filter(({ key }) => m[key] !== null).map(({ key, short }) => `${short} ${formatAmount(m[key] ?? 0)}`)
  return `${parts.join(' · ')} g`
}
