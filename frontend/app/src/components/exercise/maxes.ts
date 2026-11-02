import type { ExerciseMax, Unit } from '../../types'

// The user's 1RM and rep maxes (3RM, 5RM…) per exercise and loads relative to the 1RM. Weight units carry their size in kg
// (kg_per_unit), so a load converts between kg and lb; a load in % 1RM (percent_of_max) converts
// to a weight with the exercise's current max. The history is kept newest first; an entry is a max
// for `reps` reps (1 = the 1RM, which is what % 1RM refers to); a rep max also gives an estimated
// 1RM.

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

// The reps an entry is a max for (1 = the 1RM).
export const repsOf = (m: ExerciseMax) => m.reps ?? 1

// "1RM", "5RM".
export const repsLabel = (reps: number) => `${reps}RM`

// The entries (newest first) that beat every earlier one of the same reps: the PRs. A 5RM is only
// compared with other 5RMs.
export function prIds(entries: ExerciseMax[], unitOf: (id: string) => Unit | undefined): Set<string> {
  const ids = new Set<string>()
  const best = new Map<number, number>()
  for (const m of [...entries].reverse()) {
    const kg = inKg(m.load_value, unitOf(m.load_unit))
    if (kg !== null && kg > (best.get(repsOf(m)) ?? 0)) {
      best.set(repsOf(m), kg)
      ids.add(m.id)
    }
  }
  return ids
}

// The newest 1RM entry, or null: what loads in % 1RM are calculated from.
export const current1Rm = (entries: ExerciseMax[]): ExerciseMax | null => entries.find((m) => repsOf(m) === 1) ?? null

// The heaviest entry of each rep count (a later one on a tie), by reps: the user's best 1RM, 3RM, 5RM…
export function repMaxes(entries: ExerciseMax[], unitOf: (id: string) => Unit | undefined): ExerciseMax[] {
  const best = new Map<number, { m: ExerciseMax; kg: number }>()
  for (const m of entries) {
    const kg = inKg(m.load_value, unitOf(m.load_unit))
    if (kg === null) continue
    const have = best.get(repsOf(m))
    if (!have || kg > have.kg) best.set(repsOf(m), { m, kg })
  }
  return [...best.entries()].sort((a, b) => a[0] - b[0]).map(([, v]) => v.m)
}

// A 1RM estimated from a load lifted for several reps (Epley: weight × (1 + reps / 30)); one rep
// is the load itself.
export const estimateOneRm = (load: number, reps: number) => (reps <= 1 ? load : load * (1 + reps / 30))

// An entry's 1RM, as an estimate for a rep max (rounded to half a unit like other calculated loads).
export const oneRmOf = (m: ExerciseMax) => (repsOf(m) <= 1 ? m.load_value : roundLoad(estimateOneRm(m.load_value, repsOf(m))))

// The 1RM to calculate loads with: the real one, else the best estimate from the rep maxes (in the
// unit of the entry it comes from), marked as estimated. Null without either.
export interface EffectiveMax {
  load_value: number
  load_unit: string
  load_unit_name: string
  estimated: boolean
  reps: number // the rep max an estimate comes from; 1 for a real 1RM
}

export function effectiveMax(entries: ExerciseMax[], unitOf: (id: string) => Unit | undefined): EffectiveMax | null {
  const real = current1Rm(entries)
  if (real) return { load_value: real.load_value, load_unit: real.load_unit, load_unit_name: real.load_unit_name, estimated: false, reps: 1 }
  let best: ExerciseMax | null = null
  let bestKg = 0
  for (const m of repMaxes(entries, unitOf)) {
    const kg = inKg(oneRmOf(m), unitOf(m.load_unit)) ?? 0
    if (kg > bestKg) {
      best = m
      bestKg = kg
    }
  }
  if (!best) return null
  return { load_value: oneRmOf(best), load_unit: best.load_unit, load_unit_name: best.load_unit_name, estimated: true, reps: repsOf(best) }
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

// The rep counts a logged set is offered as a new max for: the 1RM and the usual rep maxes.
export const offerReps = [1, 2, 3, 5]

// A set offered as a new max, with the reps it is one for.
export interface MaxOffer<T> {
  set: T
  reps: number
}

// The logged sets that are new maxes worth saving: per rep count in offerReps, the heaviest set of
// exactly that many reps if it beats the user's best entry of that rep count (or there is none).
// With several singles logged that is the highest, not the first or last one. Sorted by reps.
export function newMaxOffers<T extends { reps_value: number | null; load_value: number | null; load_unit: string | null }>(
  sets: T[],
  unitOf: (id: string) => Unit | undefined,
  entries: ExerciseMax[],
): MaxOffer<T>[] {
  const best = new Map(repMaxes(entries, unitOf).map((m) => [repsOf(m), inKg(m.load_value, unitOf(m.load_unit)) ?? 0]))
  const offers = new Map<number, { set: T; kg: number }>()
  for (const s of sets) {
    const reps = s.reps_value
    if (reps === null || !offerReps.includes(reps) || s.load_value === null) continue
    const kg = inKg(s.load_value, s.load_unit ? unitOf(s.load_unit) : undefined)
    if (kg === null || kg <= 0 || kg <= (best.get(reps) ?? 0) + 1e-9) continue
    if (kg > (offers.get(reps)?.kg ?? 0)) offers.set(reps, { set: s, kg })
  }
  return [...offers.entries()].sort((a, b) => a[0] - b[0]).map(([reps, o]) => ({ set: o.set, reps }))
}

// The rep maxes (not the 1RM) as one line, e.g. "3RM 125 kg · 5RM 115 kg"; empty without.
export function repMaxText(entries: ExerciseMax[], unitOf: (id: string) => Unit | undefined): string {
  return repMaxes(entries, unitOf)
    .filter((m) => repsOf(m) > 1)
    .map((m) => `${repsLabel(repsOf(m))} ${formatLoad(m.load_value, m.load_unit_name)}`)
    .join(' · ')
}

// A load for display, e.g. "102.5 kg".
export const formatLoad = (value: number, unitName: string) => `${Number(value.toFixed(2))} ${unitName}`
