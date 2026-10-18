import type { Amount, BlockKind, ExerciseType, TrainingDetail, TrainingInput, Unit } from '../../types'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'

// Editable rows of the training form. key keeps v-for stable while rows move, since the same
// exercise may appear several times and new units and blocks have no id yet. Blocks and
// exercises both keep sets, so switching kinds doesn't lose them; toInput sends the side the
// kind uses.
export interface ExerciseRow extends Amount {
  key: number
  exercise_id: string
  name: string
  type: ExerciseType
  sets: number
  tempo: string
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
  blocks: BlockRow[]
}

// Keys only need to be unique on the page, so a module-wide counter will do.
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

// Copies just the amount fields, leaving behind unit names and the row's other fields.
const pickAmount = (a: Amount): Amount => ({
  reps_value: a.reps_value,
  reps_max: a.reps_max,
  reps_unit: a.reps_unit,
  per_side: a.per_side,
  load_value: a.load_value,
  load_max: a.load_max,
  load_unit: a.load_unit,
})

export const DEFAULT_SETS = 3

export const newBlock = (): BlockRow => ({
  key: nextKey(),
  kind: useLookupsStore().blockKinds[0]?.code ?? 'single',
  name: '',
  sets: DEFAULT_SETS,
  notes: '',
  exercises: [],
  ...emptyAmount(),
})

export const newUnit = (name: string): UnitRow => ({
  key: nextKey(),
  name,
  notes: '',
  blocks: [newBlock()],
})

// Turns a loaded training into editable rows. The side the kind doesn't use has no sets, so it
// borrows them from the other side: switching to another kind then starts from a sensible value.
export function toRows(training: TrainingDetail): UnitRow[] {
  return training.units.map((u) => ({
    key: nextKey(),
    id: u.id,
    name: u.name,
    notes: u.notes,
    blocks: u.blocks.map((b) => ({
      key: nextKey(),
      kind: b.kind,
      name: b.name,
      sets: b.sets ?? b.exercises[0]?.sets ?? DEFAULT_SETS,
      notes: b.notes,
      ...pickAmount(b),
      exercises: b.exercises.map((e) => ({
        key: nextKey(),
        exercise_id: e.exercise_id,
        name: e.name,
        type: e.type,
        sets: e.sets ?? b.sets ?? DEFAULT_SETS,
        tempo: e.tempo,
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

// Builds the request body from the rows, sending sets only on the side the block's kind uses
// (the server clears reps and load on the unused side itself).
export function toInput(fields: Omit<TrainingInput, 'units'>, units: UnitRow[]): TrainingInput {
  return {
    ...fields,
    units: units.map(({ id, name, notes, blocks }) => ({
      id,
      name,
      notes,
      blocks: blocks.map((b) => {
        const sharedSets = useLookupsStore().kindOf(b.kind)?.shared_sets ?? true
        return {
          kind: b.kind,
          name: b.name,
          sets: sharedSets ? b.sets : null,
          notes: b.notes,
          ...cleanAmount(b),
          exercises: b.exercises.map((e) => ({
            exercise_id: e.exercise_id,
            sets: sharedSets ? null : e.sets,
            tempo: e.tempo,
            notes: e.notes,
            ...cleanAmount(e),
          })),
        }
      }),
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

// Labels an exercise: just the block's letter when it's the block's only exercise, else A1, A2...
export function exerciseLabel(count: number, blockIndex: number, index: number) {
  const letter = blockLetter(blockIndex)
  return count === 1 ? letter : `${letter}${index + 1}`
}

// Describes how many exercises a kind of block takes.
export function exerciseCountRule(kind: BlockKind) {
  if (kind.max_exercises === null)
    return `A ${kind.name} block needs at least ${kind.min_exercises} exercise${kind.min_exercises === 1 ? '' : 's'}.`
  if (kind.max_exercises === kind.min_exercises)
    return `A ${kind.name} block takes exactly ${kind.min_exercises} exercise${kind.min_exercises === 1 ? '' : 's'}.`
  return `A ${kind.name} block takes ${kind.min_exercises} to ${kind.max_exercises} exercises.`
}

// Why a block can't be saved yet, if anything; mirrors the server's checks.
export function blockProblem(b: BlockRow): string {
  const kind = useLookupsStore().kindOf(b.kind)
  if (!kind) return 'Choose a block kind.'
  const n = b.exercises.length
  if (n < kind.min_exercises || (kind.max_exercises !== null && n > kind.max_exercises)) return exerciseCountRule(kind)
  if (kind.shared_sets) {
    if (!validSets(b.sets)) return 'Sets must be 1-100.'
  } else {
    const e = b.exercises.find((e) => !validSets(e.sets))
    if (e) return `${e.name}: Sets must be 1-100.`
  }

  // Reps and load are checked on the block when its kind shares them, else on every exercise.
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

const validSets = (sets: number) => Number.isInteger(sets) && sets >= 1 && sets <= 100

// Checks the reps (required) or load (optional) of an amount.
function amountProblem(a: Amount, part: AmountPart): string {
  const [value, max, unitId] =
    part === 'reps' ? [a.reps_value, a.reps_max, a.reps_unit] : [a.load_value, a.load_max, a.load_unit]
  if (value === null) return part === 'reps' ? 'Reps are required.' : ''
  const unit = useLookupsStore().unitsOf(part).find((u) => u.id === unitId)
  if (!unit) return `Choose a ${part} unit.`
  if (max !== null && max <= value) return `The upper end of the ${part} range must be higher than ${value}.`
  for (const v of [value, max]) {
    if (v !== null && !withinBounds(v, unit)) return `${part === 'reps' ? 'Reps' : 'Load'} in ${unit.name} ${boundsText(unit)}.`
  }
  return ''
}

// The column range (NUMERIC(7, 2), like the server's maxAmount) and the unit's own bounds.
const withinBounds = (v: number, u: Unit) =>
  v >= 0 && v < 100000 && (u.min_value === null || v >= u.min_value) && (u.max_value === null || v <= u.max_value)

const boundsText = (u: Unit) => `must be between ${u.min_value ?? 0} and ${u.max_value ?? 99999}`
