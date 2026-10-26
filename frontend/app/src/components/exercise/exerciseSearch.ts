import { fuzzySearch } from '../../fuzzySearch'
import { exerciseTypeLabels, type Exercise } from '../../types'

// Finds exercises by name, focus (name or alias), muscle group, joint and type.
export const searchExercises = fuzzySearch<Exercise>([
  { name: 'name', weight: 3, get: (e) => e.name },
  { name: 'focuses', weight: 2, get: (e) => (e.focuses ?? []).flatMap((f) => [f.name, f.alias]) },
  { name: 'muscle_groups', weight: 1.5, get: (e) => e.muscle_groups ?? [] },
  { name: 'joints', weight: 1.5, get: (e) => e.joints ?? [] },
  { name: 'type', weight: 1, get: (e) => exerciseTypeLabels[e.type] ?? '' },
])
