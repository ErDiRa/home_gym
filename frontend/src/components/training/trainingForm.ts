import type { ExerciseType, TrainingDetail, TrainingInput } from '../../types'

// Editable rows of the training form. key keeps v-for stable while rows move, since the same
// exercise may appear several times and new units and blocks have no id yet.
export interface ExerciseRow {
  key: number
  exercise_id: string
  name: string
  type: ExerciseType
  sets: number
  reps: string
  notes: string
}

export interface BlockRow {
  key: number
  name: string
  notes: string
  exercises: ExerciseRow[]
}

export interface UnitRow {
  key: number
  id?: string
  name: string
  notes: string
  weekdays: number[]
  dates: string[]
  blocks: BlockRow[]
}

let lastKey = 0
export const nextKey = () => ++lastKey

export const newBlock = (): BlockRow => ({ key: nextKey(), name: '', notes: '', exercises: [] })

export const newUnit = (name: string): UnitRow => ({
  key: nextKey(),
  name,
  notes: '',
  weekdays: [],
  dates: [],
  blocks: [newBlock()],
})

export function toRows(training: TrainingDetail): UnitRow[] {
  return training.units.map((u) => ({
    key: nextKey(),
    id: u.id,
    name: u.name,
    notes: u.notes,
    weekdays: [...u.weekdays],
    dates: [...u.dates],
    blocks: u.blocks.map((b) => ({
      key: nextKey(),
      name: b.name,
      notes: b.notes,
      exercises: b.exercises.map((e) => ({ ...e, key: nextKey() })),
    })),
  }))
}

export function toInput(fields: Omit<TrainingInput, 'units'>, units: UnitRow[]): TrainingInput {
  return {
    ...fields,
    units: units.map(({ id, name, notes, weekdays, dates, blocks }) => ({
      id,
      name,
      notes,
      weekdays,
      dates,
      blocks: blocks.map(({ name, notes, exercises }) => ({
        name,
        notes,
        exercises: exercises.map(({ exercise_id, sets, reps, notes }) => ({ exercise_id, sets, reps, notes })),
      })),
    })),
  }
}

// Moves the item at index one step up (-1) or down (1).
export function move<T>(list: T[], index: number, by: -1 | 1) {
  const [item] = list.splice(index, 1)
  list.splice(index + by, 0, item)
}
