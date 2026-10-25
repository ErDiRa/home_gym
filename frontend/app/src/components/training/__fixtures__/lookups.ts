import { createPinia, setActivePinia } from 'pinia'
import { useLookupsStore } from '../../../stores/lookups'
import type { Exercise } from '../../../types'

// Lookups like the seeded ones (the block kinds and units the fixtures use), in a fresh Pinia,
// for the parser tests. Returns the kind codes, as the parsers take them.
export function useTestLookups(): string[] {
  setActivePinia(createPinia())
  const lookups = useLookupsStore()
  const kind = (code: string, min: number, sets: boolean, reps = false, load = false, scored = false) => ({
    code,
    name: code,
    description: '',
    min_exercises: min,
    max_exercises: null,
    shared_sets: sets,
    shared_reps: reps,
    shared_load: load,
    scored,
  })
  lookups.blockKinds = [
    kind('single', 1, false),
    kind('superset', 2, true),
    kind('complex', 2, true, true, true),
    kind('wod', 1, true, false, false, true),
  ]
  const unit = (name: string, min: number | null = null) => ({ id: `unit-${name}`, name, min_value: min, max_value: null })
  lookups.repUnits = [unit('reps', 1), unit('s', 1), unit('cal', 1)]
  lookups.loadUnits = [unit('kg', 0), unit('% 1RM', 0)]
  return lookups.blockKinds.map((k) => k.code)
}

// An exercise as the API lists it.
export const exercise = (name: string): Exercise => ({
  id: `ex-${name}`,
  name,
  type: 'strength',
  joints: [],
  muscle_groups: [],
  focuses: [],
  description: '',
  video_urls: [],
})

// The exercises by name key, as the import resolves them.
export const byName = (...names: string[]) => new Map(names.map((n) => [n.trim().toLowerCase(), exercise(n)]))
