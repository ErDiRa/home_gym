import type { BlockKind, LoggedSet, ScoreType, TrainingSession, Unit, WorkoutResult, WorkoutSet } from '../../types'
import { convertLoad, roundLoad, type MaxLoad } from '../exercise/maxes'
import { blockLetter, exerciseLabel } from '../training/trainingForm'
import { emomRounds, wodSummary } from '../training/wod'

// The workout screen as editable cards: the session's blocks and exercises in plan order, each
// exercise with one row per planned set, filled with what was logged or else the plan. A WOD block
// has no set rows but one score for the whole block. Logged sets and scores the plan doesn't have
// (the session changed, or it was deleted) are kept apart as extras.

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
  percentPlan: { value: number; max: number | null } | null // a load planned in % 1RM (a range with max)
  newMax: LoggedSet | null // a logged single above the current 1RM, offered to be saved as the new one
  scored: boolean // in a WOD block: no set rows, the block has one score
  rows: SetRow[]
}

// The score of a WOD block as in the inputs; capped marks a "for time" not finished in the cap,
// whose score is then the reps done.
export interface ScoreDraft {
  time_seconds: number | null
  rounds: number | null
  reps: number | null
  calories: number | null
  capped: boolean
  rx: boolean
  notes: string
}

export interface WodCard {
  blockPosition: number
  scoreType: ScoreType
  intervals: number | null // of an EMOM, e.g. 10; null when unknown
  draft: ScoreDraft
  result: WorkoutResult | null // as saved once logged
  saving: boolean
}

export interface BlockCard {
  key: string
  letter: string
  title: string // the kind and the block's name, e.g. "Superset · Accessories"
  hint: string // how to go through it, e.g. "3 rounds: A1 → A2" or "AMRAP 12:00"
  notes: string
  exercises: ExerciseCard[]
  complex: boolean // its exercises are done in rounds with the same weight: one row per round, see roundsOf
  wod: WodCard | null // set for a WOD block
}

// The inputs of a score: what was logged, else empty and Rx.
export function scoreDraft(result: WorkoutResult | null): ScoreDraft {
  return {
    time_seconds: result?.time_seconds ?? null,
    rounds: result?.rounds ?? null,
    reps: result?.reps ?? null,
    calories: result?.calories ?? null,
    capped: result?.score_type === 'for_time' && result.time_seconds === null,
    rx: result?.rx ?? true,
    notes: result?.notes ?? '',
  }
}

// The values of a draft that count for the format (the server drops the others).
export function scoreValues(scoreType: ScoreType, d: ScoreDraft) {
  const v = { time_seconds: null as number | null, rounds: null as number | null, reps: null as number | null, calories: null as number | null }
  if (scoreType === 'for_time') {
    if (d.capped) v.reps = num(d.reps)
    else v.time_seconds = d.time_seconds
  } else if (scoreType === 'amrap') {
    v.rounds = num(d.rounds)
    v.reps = num(d.reps)
  } else if (scoreType === 'max_reps') v.reps = num(d.reps)
  else if (scoreType === 'max_calories') v.calories = num(d.calories)
  else v.rounds = num(d.rounds)
  return v
}

// Whether a logged score was edited since it was saved.
export function scoreChanged(wod: WodCard): boolean {
  const r = wod.result
  if (!r) return false
  const v = scoreValues(wod.scoreType, wod.draft)
  return (
    r.time_seconds !== v.time_seconds ||
    r.rounds !== v.rounds ||
    r.reps !== v.reps ||
    r.calories !== v.calories ||
    r.rx !== wod.draft.rx ||
    r.notes !== wod.draft.notes.trim()
  )
}

// One round of a complex: the set of that number of each of its exercises, logged together with
// the same reps and load (rows of sets logged beyond the plan may be missing for some).
export interface Round {
  number: number
  sets: { card: ExerciseCard; row: SetRow }[]
}

export function roundsOf(block: BlockCard): Round[] {
  const numbers = [...new Set(block.exercises.flatMap((c) => c.rows.map((r) => r.number)))].sort((a, b) => a - b)
  return numbers.map((number) => ({
    number,
    sets: block.exercises.flatMap((card) => card.rows.filter((r) => r.number === number).map((row) => ({ card, row }))),
  }))
}

// A round is logged when all its sets are, and edited when some are or were changed since.
export function roundState(round: Round): 'open' | 'logged' | 'edited' {
  const logged = round.sets.filter((s) => s.row.logged)
  if (!logged.length) return 'open'
  return logged.length === round.sets.length && !logged.some((s) => changed(s.card, s.row)) ? 'logged' : 'edited'
}

// How much of a block is logged: its sets, or a complex's rounds.
export function progressOf(block: BlockCard): { logged: number; total: number } {
  if (block.complex) {
    const rounds = roundsOf(block)
    return { logged: rounds.filter((r) => roundState(r) === 'logged').length, total: rounds.length }
  }
  const rows = block.exercises.flatMap((c) => c.rows)
  return { logged: rows.filter((r) => r.logged).length, total: rows.length }
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

// What loads convert with: the load units by id and the user's current 1RM of an exercise.
export interface LoadContext {
  unitOf: (id: string) => Unit | undefined
  maxOf: (exerciseId: string) => MaxLoad | null
}

// Builds the cards of a workout from the session's plan (null when the session is gone) and the sets
// and scores logged so far. kindOf gives a block kind by code; defaultLoadUnit is where load
// starts when the plan has none. A load planned in % 1RM is done in the unit of the exercise's
// 1RM, calculated from it, when there is one (loads); a planned load in another unit than the
// logged sets' is converted too, where it can be.
export function buildCards(
  session: TrainingSession | null,
  sets: WorkoutSet[],
  results: WorkoutResult[],
  kindOf: (code: string) => BlockKind | undefined,
  defaultLoadUnit: string | null,
  loads?: LoadContext,
): { blocks: BlockCard[]; extras: ExerciseCard[]; extraResults: WorkoutResult[] } {
  const resultByBlock = new Map(results.map((r) => [r.block_position, r]))
  const loggedByPlace = new Map<string, WorkoutSet[]>()
  for (const s of sets) {
    const key = placeKey(s.block_position, s.exercise_position)
    loggedByPlace.set(key, [...(loggedByPlace.get(key) ?? []), s])
  }

  const blocks = (session?.blocks ?? []).map((b, i): BlockCard => {
    const kind = kindOf(b.kind)
    const scored = !!kind?.scored && b.score_type !== null
    const exercises = b.exercises.map((e, j): ExerciseCard => {
      const key = placeKey(i + 1, j + 1)
      // The server clears what the block's kind shares on the exercises, so whichever side is
      // set is the plan.
      const planned = b.sets ?? e.sets ?? 1
      const reps = e.reps_value !== null ? e : b
      const load = e.load_unit !== null ? e : b
      const logged = (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id === e.exercise_id)
      loggedByPlace.set(key, (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id !== e.exercise_id))
      const planUnit = load.load_unit ? loads?.unitOf(load.load_unit) : undefined
      const max = loads?.maxOf(e.exercise_id) ?? null
      const startUnit = planUnit?.percent_of_max && max?.unit ? max.unit.id : load.load_unit
      const loadUnit = logged.find((s) => s.load_unit)?.load_unit ?? startUnit ?? defaultLoadUnit
      let plannedLoad: number | null = null
      if (load.load_value !== null && load.load_unit === loadUnit) plannedLoad = load.load_value
      else if (load.load_value !== null && loadUnit) {
        const converted = convertLoad(load.load_value, planUnit, loads?.unitOf(loadUnit), max)
        plannedLoad = converted === null ? null : roundLoad(converted)
      }

      // A WOD's exercises are done in its rounds, so they show their reps without sets, if any.
      const text: string[] = []
      if (reps.reps_value !== null || !scored) {
        const amount = `${range(reps.reps_value ?? 0, reps.reps_max)} ${reps.reps_unit_name}${reps.per_side ? ' per side' : ''}`
        text.push(scored ? amount : `${planned} × ${amount}`)
      }
      if (load.load_value !== null) text.push(`${range(load.load_value, load.load_max)} ${load.load_unit_name}`)
      else if (load.load_unit !== null) text.push(`in ${load.load_unit_name}`) // logged in this unit, e.g. "in cal"
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
        percentPlan: planUnit?.percent_of_max && load.load_value !== null ? { value: load.load_value, max: load.load_max } : null,
        newMax: null,
        scored,
        // Sets logged in a WOD block before it became one still show.
        rows: scored ? rowsFor(0, logged, null, null) : rowsFor(planned, logged, reps.reps_value, plannedLoad),
      }
    })
    const complex = !!kind?.shared_load && !scored && exercises.length > 1
    // A complex is logged in one unit; a set not logged yet takes the load another exercise was
    // logged with in that round.
    if (complex) {
      for (const card of exercises) {
        card.loadUnit = exercises[0].loadUnit
        for (const row of card.rows) {
          if (row.logged) continue
          const done = exercises.flatMap((c) => c.rows).find((r) => r.number === row.number && r.logged?.load_unit === card.loadUnit)
          if (done) row.load = done.logged!.load_value
        }
      }
    }
    const sets = b.sets ?? 0
    const result = scored ? (resultByBlock.get(i + 1) ?? null) : null
    if (scored) resultByBlock.delete(i + 1)
    return {
      key: `block-${i + 1}`,
      letter: blockLetter(i),
      title: [kind?.name ?? b.kind, b.name].filter(Boolean).join(' · '),
      // Exercises sharing sets are done in rounds, one after the other.
      hint: scored
        ? wodSummary(b, b.sets)
        : kind?.shared_sets && exercises.length > 1
          ? `${sets} ${sets === 1 ? 'round' : 'rounds'}: ${exercises.map((e) => e.label).join(' → ')}`
          : '',
      notes: b.notes,
      exercises,
      complex,
      wod: scored
        ? {
            blockPosition: i + 1,
            scoreType: b.score_type!,
            intervals: emomRounds(b),
            draft: scoreDraft(result),
            result,
            saving: false,
          }
        : null,
    }
  })
  // A score logged for a block that is no longer a WOD, or no longer there, is kept apart too.
  const extraResults = [...resultByBlock.values()]

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
      percentPlan: null,
      newMax: null,
      scored: false,
      rows: rowsFor(0, logged, null, null),
    })
  }
  extras.sort((a, b) => a.blockPosition - b.blockPosition || a.position - b.position)
  return { blocks, extras, extraResults }
}

// One line of a session card's summary: an exercise with its label, and where it sits in a
// block of several (to draw them joined), else null.
// One block of a session card: its letter, its name (its exercises when it has none) and below
// what it holds: a WOD's format and the exercises, unless the name already says just that.
export interface SummaryLine {
  key: string
  label: string
  name: string
  detail: string
}

// The session's blocks in plan order, as a session card lists them.
export function sessionSummary(session: TrainingSession): SummaryLine[] {
  return session.blocks.map((b, i) => {
    const exercises = [...new Set(b.exercises.map((e) => e.name))].join(', ')
    const name = b.name.trim() || exercises
    const parts = [b.score_type ? wodSummary(b, b.sets) : '', exercises === name ? '' : exercises]
    return { key: String(i + 1), label: blockLetter(i), name, detail: parts.filter(Boolean).join(' — ') }
  })
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
