import type { BlockKind, LoggedSet, TrainingSession, WorkoutSet } from '../../types'
import { blockLetter, exerciseLabel } from '../training/trainingForm'

// The workout screen as editable cards: the session's blocks and exercises in plan order, each
// exercise with one row per planned set, filled with what was logged or else the plan. Logged
// sets the plan doesn't have (the session changed, or it was deleted) are kept apart as extras.

// One set: the values in the inputs, and the set as saved once logged.
export interface SetRow {
  number: number
  reps: number | null
  load: number | null
  logged: LoggedSet | null
  saving: boolean
}

export interface ExerciseCard {
  key: string
  label: string // A, B1, B2...
  blockPosition: number // from 1, like the logged sets
  position: number // in the block, from 1
  exerciseId: string
  name: string
  planned: string // e.g. "3 × 8–10 reps · 60 kg"; empty for extras
  tempo: string
  notes: string
  repsUnit: string
  repsUnitName: string
  perSide: boolean
  loadUnit: string | null // the unit the load inputs are in
  rows: SetRow[]
}

export interface BlockCard {
  key: string
  letter: string
  title: string // the kind and the block's name, e.g. "Superset · Accessories"
  hint: string // how to go through it, e.g. "3 rounds: A1 → A2"
  notes: string
  exercises: ExerciseCard[]
}

// A value or a range, e.g. "8" or "8–10".
const range = (value: number, max: number | null) => (max === null ? `${value}` : `${value}–${max}`)

// Keys a logged set by the exercise's place in the session.
const placeKey = (blockPosition: number, position: number) => `${blockPosition}-${position}`

// Rows for sets 1..planned, filled with what was logged (or the planned values), followed by any
// further sets logged beyond the plan.
function rowsFor(planned: number, logged: WorkoutSet[], reps: number | null, load: number | null): SetRow[] {
  const byNumber = new Map(logged.map((s) => [s.set_number, s]))
  const count = Math.max(planned, ...logged.map((s) => s.set_number), 0)
  const rows: SetRow[] = []
  for (let n = 1; n <= count; n++) {
    const set = byNumber.get(n) ?? null
    // Sets beyond the plan that weren't logged (a gap in the numbers) are skipped.
    if (n > planned && !set) continue
    rows.push({ number: n, reps: set ? set.reps_value : reps, load: set ? set.load_value : load, logged: set, saving: false })
  }
  return rows
}

// Builds the cards of a workout from the session's plan (null when the session is gone) and the sets
// logged so far. kindOf gives a block kind by code; defaultLoadUnit is where load starts when
// the plan has none.
export function buildCards(
  session: TrainingSession | null,
  sets: WorkoutSet[],
  kindOf: (code: string) => BlockKind | undefined,
  defaultLoadUnit: string | null,
): { blocks: BlockCard[]; extras: ExerciseCard[] } {
  const loggedByPlace = new Map<string, WorkoutSet[]>()
  for (const s of sets) {
    const key = placeKey(s.block_position, s.exercise_position)
    loggedByPlace.set(key, [...(loggedByPlace.get(key) ?? []), s])
  }

  const blocks = (session?.blocks ?? []).map((b, i): BlockCard => {
    const exercises = b.exercises.map((e, j): ExerciseCard => {
      const key = placeKey(i + 1, j + 1)
      // The server clears what the block's kind shares on the exercises, so whichever side is
      // set is the plan.
      const planned = b.sets ?? e.sets ?? 1
      const reps = e.reps_value !== null ? e : b
      const load = e.load_value !== null ? e : b
      const logged = (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id === e.exercise_id)
      loggedByPlace.set(key, (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id !== e.exercise_id))
      const loadUnit = logged.find((s) => s.load_unit)?.load_unit ?? load.load_unit ?? defaultLoadUnit
      const plannedLoad = load.load_unit === loadUnit ? load.load_value : null

      const text = [`${planned} × ${range(reps.reps_value ?? 0, reps.reps_max)} ${reps.reps_unit_name}${reps.per_side ? ' per side' : ''}`]
      if (load.load_value !== null) text.push(`${range(load.load_value, load.load_max)} ${load.load_unit_name}`)
      return {
        key,
        label: exerciseLabel(b.exercises.length, i, j),
        blockPosition: i + 1,
        position: j + 1,
        exerciseId: e.exercise_id,
        name: e.name,
        planned: text.join(' · '),
        tempo: e.tempo,
        notes: e.notes,
        repsUnit: reps.reps_unit ?? '',
        repsUnitName: reps.reps_unit_name,
        perSide: reps.per_side,
        loadUnit,
        rows: rowsFor(planned, logged, reps.reps_value, plannedLoad),
      }
    })
    const kind = kindOf(b.kind)
    const sets = b.sets ?? 0
    return {
      key: `block-${i + 1}`,
      letter: blockLetter(i),
      title: [kind?.name ?? b.kind, b.name].filter(Boolean).join(' · '),
      // Exercises sharing sets are done in rounds, one after the other.
      hint:
        kind?.shared_sets && exercises.length > 1
          ? `${sets} ${sets === 1 ? 'round' : 'rounds'}: ${exercises.map((e) => e.label).join(' → ')}`
          : '',
      notes: b.notes,
      exercises,
    }
  })

  // Whatever is left was logged at a place the plan no longer has.
  const extras: ExerciseCard[] = []
  for (const [key, logged] of loggedByPlace) {
    if (!logged.length) continue
    const first = logged[0]
    extras.push({
      key: `extra-${key}`,
      label: exerciseLabel(2, first.block_position - 1, first.exercise_position - 1),
      blockPosition: first.block_position,
      position: first.exercise_position,
      exerciseId: first.exercise_id,
      name: first.exercise_name,
      planned: '',
      tempo: '',
      notes: '',
      repsUnit: first.reps_unit,
      repsUnitName: first.reps_unit_name,
      perSide: first.per_side,
      loadUnit: logged.find((s) => s.load_unit)?.load_unit ?? defaultLoadUnit,
      rows: rowsFor(0, logged, null, null),
    })
  }
  extras.sort((a, b) => a.blockPosition - b.blockPosition || a.position - b.position)
  return { blocks, extras }
}

// One line of a session card's summary: an exercise with its label, and where it sits in a
// block of several (to draw them joined), else null.
export interface SummaryLine {
  key: string
  label: string
  name: string
  group: 'first' | 'middle' | 'last' | null
}

// The session's exercises in plan order, as a session card lists them.
export function sessionSummary(session: TrainingSession): SummaryLine[] {
  return session.blocks.flatMap((b, i) =>
    b.exercises.map((e, j) => ({
      key: `${i + 1}-${j + 1}`,
      label: exerciseLabel(b.exercises.length, i, j),
      name: e.name,
      group: b.exercises.length === 1 ? null : j === 0 ? 'first' : j === b.exercises.length - 1 ? 'last' : 'middle',
    })),
  )
}

// A number input's value: v-model.number leaves '' in an emptied input, which counts as none.
export const num = (v: unknown): number | null => (typeof v === 'number' && Number.isFinite(v) ? v : null)

// Whether a logged row was edited since it was saved.
export function changed(card: ExerciseCard, row: SetRow): boolean {
  const s = row.logged
  if (!s) return false
  const load = num(row.load)
  return s.reps_value !== num(row.reps) || s.load_value !== load || (load !== null && s.load_unit !== card.loadUnit)
}
