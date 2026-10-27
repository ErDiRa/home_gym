import type {
  Amount,
  BlockKind,
  ExerciseType,
  Score,
  TrainingBlock,
  TrainingDetail,
  TrainingInput,
  TrainingSessionInput,
  Unit,
} from '../../types'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import { scoreFormat, scoreProblem } from './wod'

// Editable rows of the training and session forms. key keeps v-for stable while rows move, since
// the same exercise may appear several times and new sessions and blocks have no id yet. Blocks and
// exercises both keep sets, so switching kinds doesn't lose them; toInput sends the side the
// kind uses. A WOD block keeps its rounds (optional) apart from the sets, and its score.
export interface ExerciseRow extends Amount {
  key: number
  exercise_id: string
  name: string
  type: ExerciseType
  sets: number
  tempo: string
  notes: string
}

export interface BlockRow extends Amount, Score {
  key: number
  kind: string
  name: string
  sets: number
  rounds: number | null // of a scored kind (a WOD); null = not fixed
  notes: string
  exercises: ExerciseRow[]
}

export interface SessionRow {
  key: number
  id?: string // a training's saved session; never set for library templates
  name: string
  notes: string
  blocks: BlockRow[]
}

// A planned session of a week: the session row it points at, by key (new sessions have no id).
export interface SlotRow {
  key: number
  id?: string // saved slots keep their id, so the workouts done of them stay linked
  sessionKey: number
}

export interface WeekRow {
  key: number
  slots: SlotRow[]
}

// At most this many sessions a week, like the server.
export const MAX_SLOTS = 7

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
  load_bodyweight: false,
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
  load_bodyweight: a.load_bodyweight,
})

export const DEFAULT_SETS = 3

export const emptyScore = (): Score => ({ score_type: null, time_seconds: null, interval_seconds: null })

export const newBlock = (): BlockRow => ({
  key: nextKey(),
  kind: useLookupsStore().blockKinds[0]?.code ?? 'single',
  name: '',
  sets: DEFAULT_SETS,
  rounds: null,
  notes: '',
  exercises: [],
  ...emptyAmount(),
  ...emptyScore(),
})

export const newSession = (name: string): SessionRow => ({
  key: nextKey(),
  name,
  notes: '',
  blocks: [newBlock()],
})

export const newWeek = (slots: SlotRow[] = []): WeekRow => ({ key: nextKey(), slots })

// Turns a loaded session into an editable row: with its id for a training's own session,
// without one for a copy (e.g. a library template added to a training).
export function toSessionRow(
  s: { id?: string; name: string; notes: string; blocks: TrainingBlock[] },
  keepId: boolean,
): SessionRow {
  return { key: nextKey(), id: keepId ? s.id : undefined, name: s.name, notes: s.notes, blocks: toBlockRows(s.blocks) }
}

// Turns a loaded training into editable sessions and weeks (one row per week).
export function toRows(training: TrainingDetail): { sessions: SessionRow[]; weeks: WeekRow[] } {
  const sessions = training.sessions.map((s) => toSessionRow(s, true))
  const keyOf = new Map(training.sessions.map((s, i) => [s.id, sessions[i].key]))
  const weeks = training.schedule.map((slots) =>
    newWeek(slots.map((slot) => ({ key: nextKey(), id: slot.id, sessionKey: keyOf.get(slot.session_id)! }))),
  )
  return { sessions, weeks }
}

// The side the kind doesn't use has no sets, so it borrows them from the other side: switching
// to another kind then starts from a sensible value.
function toBlockRows(blocks: TrainingBlock[]): BlockRow[] {
  return blocks.map((b) => ({
    key: nextKey(),
    kind: b.kind,
    name: b.name,
    sets: b.sets ?? b.exercises[0]?.sets ?? DEFAULT_SETS,
    rounds: b.score_type ? b.sets : null,
    notes: b.notes,
    ...pickAmount(b),
    score_type: b.score_type,
    time_seconds: b.time_seconds,
    interval_seconds: b.interval_seconds,
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
  }))
}

// cleanAmount drops what an empty value leaves behind (a preselected unit, a lone max). A load
// unit stays without a value: the unit the sets are logged in, e.g. calories on a machine.
function cleanAmount(a: Amount): Amount {
  const out = pickAmount(a)
  if (out.reps_value === null) Object.assign(out, { reps_max: null, reps_unit: null, per_side: false })
  if (out.load_value === null) out.load_max = null
  return out
}

// The score of a block as sent: only for a scored kind, the interval only for an EMOM.
function scoreInput(b: BlockRow, scored: boolean): Score {
  if (!scored) return emptyScore()
  return {
    score_type: b.score_type,
    time_seconds: b.time_seconds,
    interval_seconds: scoreFormat(b.score_type)?.hasInterval ? b.interval_seconds : null,
  }
}

// Builds a session of the request body, sending sets only on the side the block's kind uses
// (the server clears reps and load on the unused side itself); a WOD's sets are its rounds.
export function toSessionInput({ id, name, notes, blocks }: SessionRow): TrainingSessionInput {
  return {
    id,
    name,
    notes,
    blocks: blocks.map((b) => {
      const kind = useLookupsStore().kindOf(b.kind)
      const sharedSets = kind?.shared_sets ?? true
      const scored = kind?.scored ?? false
      return {
        kind: b.kind,
        name: b.name,
        sets: scored ? b.rounds : sharedSets ? b.sets : null,
        notes: b.notes,
        ...cleanAmount(b),
        ...scoreInput(b, scored),
        exercises: b.exercises.map((e) => ({
          exercise_id: e.exercise_id,
          sets: sharedSets ? null : e.sets,
          tempo: e.tempo,
          notes: e.notes,
          ...cleanAmount(e),
        })),
      }
    }),
  }
}

// Builds the training's request body; the slots point at their sessions by index.
export function toInput(
  fields: Omit<TrainingInput, 'sessions' | 'schedule'>,
  sessions: SessionRow[],
  weeks: WeekRow[],
): TrainingInput {
  const index = new Map(sessions.map((s, i) => [s.key, i]))
  return {
    ...fields,
    sessions: sessions.map(toSessionInput),
    schedule: weeks.map((w) => w.slots.map((slot) => ({ id: slot.id, session: index.get(slot.sessionKey)! }))),
  }
}

// A loaded training as the request body that saves it unchanged, e.g. to move it into another
// cycle. Needs the training lookups loaded (the block kinds decide which side keeps the sets).
export function inputOf(training: TrainingDetail): TrainingInput {
  const { sessions, weeks } = toRows(training)
  const { cycle_id, name, description, active, start_date } = training
  return toInput({ cycle_id, name, description, active, start_date, weeks: training.weeks }, sessions, weeks)
}

// Why a session can't be saved yet, if anything; mirrors the server's checks.
export function sessionProblem(s: SessionRow): string {
  if (!s.name.trim()) return 'Every session needs a name.'
  for (const [b, block] of s.blocks.entries()) {
    const p = blockProblem(block)
    if (p) return `${s.name}, block ${blockLetter(b)}: ${p}`
  }
  return ''
}

// Moves the item at index one step up (-1) or down (1).
export function move<T>(list: T[], index: number, by: -1 | 1) {
  const [item] = list.splice(index, 1)
  list.splice(index + by, 0, item)
}

// Labels a block by its position in the session: A, B, ... Z, then 27, 28...
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
  if (kind.scored) {
    const problem = scoreProblem(b)
    if (problem) return problem
    if (b.rounds !== null && !validSets(b.rounds)) return 'Rounds must be 1-100.'
  } else if (kind.shared_sets) {
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
      // In a WOD an exercise may go without reps, e.g. max calories on the rower.
      const problem = amountProblem(a, part, kind.scored && !shared)
      if (problem) return shared ? problem : `${(a as ExerciseRow).name}: ${problem}`
    }
  }
  return ''
}

const validSets = (sets: number) => Number.isInteger(sets) && sets >= 1 && sets <= 100

// Checks the reps (required unless optional) or load (optional) of an amount.
function amountProblem(a: Amount, part: AmountPart, optional = false): string {
  const [value, max, unitId] =
    part === 'reps' ? [a.reps_value, a.reps_max, a.reps_unit] : [a.load_value, a.load_max, a.load_unit]
  if (value === null) return part === 'reps' && !optional ? 'Reps are required.' : ''
  const unit = useLookupsStore().unitsOf(part).find((u) => u.id === unitId)
  if (!unit) return `Choose a ${part} unit.`
  if (part === 'load' && a.load_bodyweight && !unit.kg_per_unit) return 'Weight added to bodyweight needs a weight unit such as kg or lb.'
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
