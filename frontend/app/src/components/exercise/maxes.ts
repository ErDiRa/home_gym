import type { ExerciseMax, Unit } from '../../types'

// The user's 1RM per exercise and loads relative to it. Weight units carry their size in kg
// (kg_per_unit), so a load converts between kg and lb; a load in % 1RM (percent_of_max) converts
// to a weight with the exercise's current max. The history is kept newest first; its first entry
// is the current max.

// A max as a load: its value in its unit.
export interface MaxLoad {
  value: number
  unit: Unit | undefined
}

// Newest first: by the day it was lifted, then by when it was entered.
export function newestFirst(entries: ExerciseMax[]): ExerciseMax[] {
  return [...entries].sort((a, b) => b.achieved_on.localeCompare(a.achieved_on) || b.created_at.localeCompare(a.created_at))
}

// A load in kg; null unless the unit is a weight unit.
export function inKg(value: number, unit: Unit | undefined): number | null {
  return unit?.kg_per_unit ? value * unit.kg_per_unit : null
}

// The entries (newest first) that beat every earlier one: the PRs.
export function prIds(entries: ExerciseMax[], unitOf: (id: string) => Unit | undefined): Set<string> {
  const ids = new Set<string>()
  let best = 0
  for (const m of [...entries].reverse()) {
    const kg = inKg(m.load_value, unitOf(m.load_unit))
    if (kg !== null && kg > best) {
      best = kg
      ids.add(m.id)
    }
  }
  return ids
}

// A load converted from one unit to another: between weight units by their size, from or to
// % 1RM with the max. Null when it can't be converted (e.g. RPE, or % 1RM without a max).
export function convertLoad(value: number, from: Unit | undefined, to: Unit | undefined, max: MaxLoad | null): number | null {
  if (!from || !to) return null
  if (from.id === to.id) return value
  const maxKg = max ? inKg(max.value, max.unit) : null
  const kg = from.kg_per_unit ? value * from.kg_per_unit : from.percent_of_max && maxKg ? (value / 100) * maxKg : null
  if (kg === null) return null
  if (to.kg_per_unit) return kg / to.kg_per_unit
  if (to.percent_of_max && maxKg) return (kg / maxKg) * 100
  return null
}

// A calculated load rounded to half a unit (e.g. 84.5 kg), as a bar can be loaded.
export const roundLoad = (v: number) => Math.round(v * 2) / 2

// Whether a logged set is a new 1RM worth saving: one rep with a weight above the current max
// (or there is none yet).
export function isNewMax(reps: number | null, load: number | null, unit: Unit | undefined, max: MaxLoad | null): boolean {
  if (reps !== 1 || load === null) return false
  const kg = inKg(load, unit)
  if (kg === null || kg <= 0) return false
  const maxKg = max ? inKg(max.value, max.unit) : null
  return maxKg === null || kg > maxKg + 1e-9
}

// A load for display, e.g. "102.5 kg".
export const formatLoad = (value: number, unitName: string) => `${Number(value.toFixed(2))} ${unitName}`
