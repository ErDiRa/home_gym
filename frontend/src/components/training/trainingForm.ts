import type { Amount, BlockKind, ExerciseType, TrainingDetail, TrainingInput, Unit } from '../../types'
import { blockKinds, kindOf, unitsOf, type AmountPart } from './lookups'

// Editable rows of the training form. key keeps v-for stable while rows move, since the same
// exercise may appear several times and new units and blocks have no id yet.
export interface ExerciseRow extends Amount {
  key: number
  exercise_id: string
  name: string
  type: ExerciseType
  notes: string
}

export interface BlockRow extends Amount {
  key: number
  kind: string
  name: string
  sets: number
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

export const emptyAmount = (): Amount => ({
  reps_value: null,
  reps_max: null,
  reps_unit: null,
  per_side: false,
  load_value: null,
  load_max: null,
  load_unit: null,
})

const pickAmount = (a: Amount): Amount => ({
  reps_value: a.reps_value,
  reps_max: a.reps_max,
  reps_unit: a.reps_unit,
  per_side: a.per_side,
  load_value: a.load_value,
  load_max: a.load_max,
  load_unit: a.load_unit,
})

export const newBlock = (): BlockRow => ({
  key: nextKey(),
  kind: blockKinds.value[0]?.code ?? 'single',
  name: '',
  sets: 3,
  notes: '',
  exercises: [],
  ...emptyAmount(),
})

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
      kind: b.kind,
      name: b.name,
      sets: b.sets,
      notes: b.notes,
      ...pickAmount(b),
      exercises: b.exercises.map((e) => ({
        key: nextKey(),
        exercise_id: e.exercise_id,
        name: e.name,
        type: e.type,
        notes: e.notes,
        ...pickAmount(e),
      })),
    })),
  }))
}

// cleanAmount drops what an empty value leaves behind (a preselected unit, a lone max).
function cleanAmount(a: Amount): Amount {
  const out = pickAmount(a)
  if (out.reps_value === null) Object.assign(out, { reps_max: null, reps_unit: null, per_side: false })
  if (out.load_value === null) Object.assign(out, { load_max: null, load_unit: null })
  return out
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
      blocks: blocks.map((b) => ({
        kind: b.kind,
        name: b.name,
        sets: b.sets,
        notes: b.notes,
        ...cleanAmount(b),
        exercises: b.exercises.map((e) => ({ exercise_id: e.exercise_id, notes: e.notes, ...cleanAmount(e) })),
      })),
    })),
  }
}

// Moves the item at index one step up (-1) or down (1).
export function move<T>(list: T[], index: number, by: -1 | 1) {
  const [item] = list.splice(index, 1)
  list.splice(index + by, 0, item)
}

// Labels a block by its position in the unit: A, B, ... Z, then 27, 28...
export const blockLetter = (index: number) => (index < 26 ? String.fromCharCode(65 + index) : String(index + 1))

// Labels an exercise: just the block's letter when the kind takes one exercise, else A1, A2...
export function exerciseLabel(kind: BlockKind | undefined, blockIndex: number, index: number) {
  const letter = blockLetter(blockIndex)
  return kind?.max_exercises === 1 ? letter : `${letter}${index + 1}`
}

// Describes how many exercises a kind of block takes.
export function exerciseCountRule(kind: BlockKind) {
  if (kind.max_exercises === null) return `A ${kind.name} block needs at least ${kind.min_exercises} exercises.`
  if (kind.max_exercises === kind.min_exercises)
    return `A ${kind.name} block takes exactly ${kind.min_exercises} exercise${kind.min_exercises === 1 ? '' : 's'}.`
  return `A ${kind.name} block takes ${kind.min_exercises} to ${kind.max_exercises} exercises.`
}

// Why a block can't be saved yet, if anything; mirrors the server's checks.
export function blockProblem(b: BlockRow): string {
  const kind = kindOf(b.kind)
  if (!kind) return 'Choose a block kind.'
  if (!(b.sets >= 1 && b.sets <= 100)) return 'Sets must be 1-100.'
  const n = b.exercises.length
  if (n < kind.min_exercises || (kind.max_exercises !== null && n > kind.max_exercises)) return exerciseCountRule(kind)

  const parts: [AmountPart, boolean][] = [
    ['reps', kind.shared_reps],
    ['load', kind.shared_load],
  ]
  for (const [part, shared] of parts) {
    const owners = shared ? [b] : b.exercises
    for (const a of owners) {
      const problem = amountProblem(a, part)
      if (problem) return shared ? problem : `${(a as ExerciseRow).name}: ${problem}`
    }
  }
  return ''
}

// Checks the reps (required) or load (optional) of an amount.
function amountProblem(a: Amount, part: AmountPart): string {
  const [value, max, unitId] =
    part === 'reps' ? [a.reps_value, a.reps_max, a.reps_unit] : [a.load_value, a.load_max, a.load_unit]
  if (value === null) return part === 'reps' ? 'Reps are required.' : ''
  const unit = unitsOf(part).value.find((u) => u.id === unitId)
  if (!unit) return `Choose a ${part} unit.`
  if (max !== null && max <= value) return `The upper end of the ${part} range must be higher than ${value}.`
  for (const v of [value, max]) {
    if (v !== null && !withinBounds(v, unit)) return `${part === 'reps' ? 'Reps' : 'Load'} in ${unit.name} ${boundsText(unit)}.`
  }
  return ''
}

const withinBounds = (v: number, u: Unit) =>
  v >= 0 && v < 100000 && (u.min_value === null || v >= u.min_value) && (u.max_value === null || v <= u.max_value)

const boundsText = (u: Unit) => `must be between ${u.min_value ?? 0} and ${u.max_value ?? 99999}`
