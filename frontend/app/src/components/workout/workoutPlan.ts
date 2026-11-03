import type { BlockKind, ExerciseNote, IntensityMetric, LoggedSet, ScoreType, TrainingSession, Unit, WorkoutResult, WorkoutSet } from '../../types'
import { intensityText } from '../exercise/intensity'
import { convertLoad, roundLoad, type MaxLoad, type MaxOffer } from '../exercise/maxes'
import { blockLetter, exerciseLabel } from '../training/trainingForm'
import { emomRounds, wodSummary } from '../training/wod'
import { clampProgress, noProgress, progressEnabled, progressOfReps, progressReps, type Progress } from './wodProgress'

// The workout screen as editable cards: the session's blocks and exercises in plan order, each
// exercise with one row per planned set, filled with what was logged or else the plan. A WOD block
// has no set rows but one score for the whole block. Logged sets and scores the plan doesn't have
// (the session changed, or it was deleted) are kept apart as extras.

// One set: the values in the inputs, and the set as saved once logged.
export interface SetRow {
  number: number
  reps: number | null
  load: number | null
  intensity: number | null // the average of an endurance exercise, in the card's intensityMetric
  logged: WorkoutSet | LoggedSet | null // as loaded with the workout, or as just logged
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
  bodyweight: boolean // done on bodyweight: the load inputs hold the weight added to it, if any
  showLoad: boolean // whether the load inputs show; on bodyweight only once weight is added
  endurance: boolean // an erg, a run…: each set also takes its average (watts, cal/h or a pace)
  intensityMetric: IntensityMetric // what the average inputs are in
  percentPlan: { value: number; max: number | null } | null // a load planned in % 1RM (a range with max)
  newMaxes: MaxOffer<WorkoutSet | LoggedSet>[] // logged sets that beat the user's 1RM or a rep max (3RM, 5RM…), offered to be saved
  lastLoad: MaxLoad | null // the load last logged in another workout, suggested when the plan has none
  scored: boolean // in a WOD block: no set rows, the block has one score
  rows: SetRow[]
  note: ExerciseNote | null // the user's note on it in this workout, see attachNotes
}

// The score of a WOD block as in the inputs; capped marks a "for time" not finished in the cap,
// whose score is then the reps done.
export interface ScoreDraft {
  time_seconds: number | null
  rounds: number | null
  reps: number | null
  calories: number | null
  capped: boolean
  progress: Progress // how far a capped WOD got; the reps are calculated from it
  notes: string
}

// One line of a WOD in plan order, e.g. "18 Hang Cleans"; first marks an exercise's first line,
// which carries its rating and video buttons.
export interface WodLine {
  key: string
  exerciseId: string
  name: string
  reps: string // e.g. "18" or "8–10 per side", with the unit unless it is plain reps
  load: string
  first: boolean
  repsValue: number | null // the plain amount, a range with its lower value
  repsUnit: string
}

export interface WodCard {
  blockPosition: number
  scoreType: ScoreType
  intervals: number | null // of an EMOM, e.g. 10; null when unknown
  rounds: number // the block's rounds (sets), at least 1
  lines: WodLine[]
  draft: ScoreDraft
  result: WorkoutResult | null // as saved once logged
  partsTotal: number | null // the sum of the reps or calories logged per movement (see partsUnit), null without
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

// The inputs of a score: what was logged, else empty.
export function scoreDraft(result: WorkoutResult | null, lines: WodLine[] = [], rounds = 1): ScoreDraft {
  const capped = result?.score_type === 'for_time' && result.time_seconds === null
  return {
    // A saved capped score shows as the progress that makes it, where the plan allows.
    progress: capped && progressEnabled(lines) ? progressOfReps(lines, rounds, result?.reps ?? 0) : { ...noProgress },
    time_seconds: result?.time_seconds ?? null,
    rounds: result?.rounds ?? null,
    reps: result?.reps ?? null,
    calories: result?.calories ?? null,
    capped,
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
  else {
    v.rounds = num(d.rounds)
    v.reps = num(d.reps) // an EMOM's total, from the reps logged per movement
  }
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

// How much of a block is logged: its sets, or a complex's rounds. A WOD counts by its score (an
// EMOM's reps per movement are logged as sets, but not planned as such).
export function progressOf(block: BlockCard): { logged: number; total: number } {
  if (block.wod) return { logged: 0, total: 0 }
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
function rowsFor(planned: number, logged: WorkoutSet[], reps: number | null, load: number | null, intensity: number | null = null): SetRow[] {
  const byNumber = new Map(logged.map((s) => [s.set_number, s]))
  const count = Math.max(planned, ...logged.map((s) => s.set_number), 0)
  const rows: SetRow[] = []
  for (let n = 1; n <= count; n++) {
    const set = byNumber.get(n) ?? null
    // Sets beyond the plan that weren't logged (a gap in the numbers) are skipped.
    if (n > planned && !set) continue
    rows.push({
      number: n,
      reps: set ? set.reps_value : reps,
      load: set ? set.load_value : load,
      intensity: set ? set.intensity_value : intensity,
      logged: set,
      saving: false,
    })
  }
  return rows
}

// What loads convert with: the load units by id and the user's current 1RM of an exercise; lastOf
// gives the load last logged of an exercise, if any, and weightUnit the unit weight added to
// bodyweight starts in (kg or lb).
export interface LoadContext {
  unitOf: (id: string) => Unit | undefined
  maxOf: (exerciseId: string) => MaxLoad | null
  lastOf?: (exerciseId: string) => MaxLoad | null
  weightUnit?: string | null
}

// Builds the cards of a workout from the session's plan (null when the session is gone) and the sets
// and scores logged so far. kindOf gives a block kind by code; defaultLoadUnit is where load
// starts when the plan has none. A load planned in % 1RM is done in the unit of the exercise's
// 1RM, calculated from it, when there is one (loads); a planned load in another unit than the
// logged sets' is converted too, where it can be. Without a planned load (or with only its unit) the
// load last logged of the exercise is suggested, and its unit is where load starts. On bodyweight
// the load inputs only show with weight added to it (planned or logged), in a weight unit. An
// endurance exercise's sets also take their average, prefilled with the planned target, in the
// metric logged, else planned, else watts (which every machine shows).
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
    // What a WOD shows of each exercise, to group the repeated ones into movements.
    const parts: { amount: string; repsValue: number | null; repsUnit: string; weight: string; plannedLoad: number | null; loadUnit: string | null; bodyweight: boolean }[] = []
    const exercises = b.exercises.map((e, j): ExerciseCard => {
      const key = placeKey(i + 1, j + 1)
      // The server clears what the block's kind shares on the exercises, so whichever side is
      // set is the plan.
      const planned = b.sets ?? e.sets ?? 1
      const reps = e.reps_value !== null ? e : b
      const load = e.load_unit !== null || e.load_bodyweight ? e : b
      const logged = (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id === e.exercise_id)
      const bodyweight = load.load_bodyweight || logged.some((s) => s.bodyweight)
      loggedByPlace.set(key, (loggedByPlace.get(key) ?? []).filter((s) => s.exercise_id !== e.exercise_id))
      const planUnit = load.load_unit ? loads?.unitOf(load.load_unit) : undefined
      const max = loads?.maxOf(e.exercise_id) ?? null
      const startUnit = planUnit?.percent_of_max && max?.unit ? max.unit.id : load.load_unit
      const last = load.load_value === null && !bodyweight ? (loads?.lastOf?.(e.exercise_id) ?? null) : null
      const loadUnit =
        logged.find((s) => s.load_unit)?.load_unit ??
        startUnit ??
        last?.unit?.id ??
        (bodyweight ? (loads?.weightUnit ?? defaultLoadUnit) : defaultLoadUnit)
      let plannedLoad: number | null = null
      if (load.load_value !== null && load.load_unit === loadUnit) plannedLoad = load.load_value
      else if (load.load_value !== null && loadUnit) {
        const converted = convertLoad(load.load_value, planUnit, loads?.unitOf(loadUnit), max)
        plannedLoad = converted === null ? null : roundLoad(converted)
      }

      // A WOD's exercises are done in its rounds, so they show their reps without sets, if any.
      const text: string[] = []
      parts[j] = {
        amount: reps.reps_value !== null ? `${range(reps.reps_value, reps.reps_max)}${reps.per_side ? '/side' : ''}` : '',
        repsValue: reps.reps_value,
        repsUnit: reps.reps_unit_name,
        weight: '',
        plannedLoad,
        loadUnit,
        bodyweight,
      }
      if (reps.reps_value !== null || !scored) {
        const amount = `${range(reps.reps_value ?? 0, reps.reps_max)} ${reps.reps_unit_name}${reps.per_side ? ' per side' : ''}`
        text.push(scored ? amount : `${planned} × ${amount}`)
      }
      const weight = load.load_value !== null ? `${range(load.load_value, load.load_max)} ${load.load_unit_name}` : ''
      parts[j].weight = bodyweight ? (weight ? `BW + ${weight}` : 'BW') : weight
      if (bodyweight) text.push(weight ? `bodyweight + ${weight}` : 'bodyweight')
      else if (weight) text.push(weight)
      else if (load.load_unit !== null) text.push(`in ${load.load_unit_name}`) // logged in this unit, e.g. "in cal"
      const endurance = e.type === 'endurance' || logged.some((s) => s.intensity_metric)
      const target = endurance ? intensityText(e) : ''
      if (target) text.push(target)
      const intensityMetric = logged.find((s) => s.intensity_metric)?.intensity_metric ?? e.intensity_metric ?? 'watts'
      const plannedIntensity = e.intensity_metric === intensityMetric ? e.intensity_value : null
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
        bodyweight,
        showLoad: !bodyweight || load.load_value !== null || logged.some((s) => s.load_value !== null),
        endurance,
        intensityMetric,
        percentPlan: planUnit?.percent_of_max && load.load_value !== null && !bodyweight ? { value: load.load_value, max: load.load_max } : null,
        newMaxes: [],
        lastLoad: scored ? null : last,
        scored,
        // Sets logged in a WOD block before it became one still show.
        rows: scored ? rowsFor(0, logged, null, null) : rowsFor(planned, logged, reps.reps_value, plannedLoad, plannedIntensity),
        note: null,
      }
    })
    const complex = !!kind?.shared_load && !scored && exercises.length > 1
    // A complex is logged in one unit; a set not logged yet takes the load another exercise was
    // logged with in that round.
    if (complex) {
      const showLoad = exercises.some((c) => c.showLoad)
      for (const card of exercises) {
        card.loadUnit = exercises[0].loadUnit
        card.bodyweight = exercises[0].bodyweight
        card.showLoad = showLoad
        for (const row of card.rows) {
          if (row.logged) continue
          const done = exercises.flatMap((c) => c.rows).find((r) => r.number === row.number && r.logged?.load_unit === card.loadUnit)
          if (done) row.load = done.logged!.load_value
        }
      }
    }
    const lines = scored ? wodLines(exercises, parts) : []
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
      wod: scored ? wodCard(i + 1, b.score_type!, emomRounds(b), Math.max(b.sets ?? 1, 1), lines, result, exercises) : null,
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
      bodyweight: first.bodyweight,
      showLoad: !first.bodyweight || logged.some((s) => s.load_value !== null),
      endurance: logged.some((s) => s.intensity_metric),
      intensityMetric: logged.find((s) => s.intensity_metric)?.intensity_metric ?? 'watts',
      percentPlan: null,
      newMaxes: [],
      lastLoad: null,
      scored: false,
      rows: rowsFor(0, logged, null, null),
      note: null,
    })
  }
  extras.sort((a, b) => a.blockPosition - b.blockPosition || a.position - b.position)
  return { blocks, extras, extraResults }
}

// The score card of a WOD block; one with parts logged per movement is scored by them (a logged
// score was saved from them already).
function wodCard(
  blockPosition: number,
  scoreType: ScoreType,
  intervals: number | null,
  rounds: number,
  lines: WodLine[],
  result: WorkoutResult | null,
  exercises: ExerciseCard[],
): WodCard {
  const wod: WodCard = { blockPosition, scoreType, intervals, rounds, lines, draft: scoreDraft(result, lines, rounds), result, partsTotal: null, saving: false }
  if (!partsUnit(wod)) return wod
  if (result) wod.partsTotal = partsTotalOf(wod, exercises)
  else fillPartsScore(wod, exercises)
  return wod
}

// The exercises of a WOD block in plan order, one line each.
function wodLines(cards: ExerciseCard[], parts: { amount: string; weight: string; repsValue: number | null; repsUnit: string }[]): WodLine[] {
  const seen = new Set<string>()
  return cards.map((c, j) => {
    const first = !seen.has(c.exerciseId)
    seen.add(c.exerciseId)
    const unit = c.repsUnitName && c.repsUnitName !== 'reps' ? ` ${c.repsUnitName}` : ''
    return { key: c.key, exerciseId: c.exerciseId, name: c.name, reps: parts[j].amount ? `${parts[j].amount}${unit}` : '', load: parts[j].weight, first, repsValue: parts[j].repsValue, repsUnit: parts[j].repsUnit }
  })
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

// The suggested load of a card's sets in the unit of its inputs, e.g. 60 for the 60 kg logged last
// time; null when there is none or it doesn't convert. max is the 1RM, for % 1RM.
export function suggestedLoad(card: ExerciseCard, unitOf: (id: string) => Unit | undefined, max: MaxLoad | null): number | null {
  const last = card.lastLoad
  if (!last || !card.loadUnit) return null
  const converted = convertLoad(last.value, last.unit, unitOf(card.loadUnit), max)
  return converted === null ? null : roundLoad(converted)
}

// Whether a logged row was edited since it was saved.
export function changed(card: ExerciseCard, row: SetRow): boolean {
  const s = row.logged
  if (!s) return false
  const load = num(row.load)
  const intensity = card.endurance ? num(row.intensity) : null
  return (
    s.reps_value !== num(row.reps) ||
    s.load_value !== load ||
    (load !== null && s.load_unit !== card.loadUnit) ||
    s.intensity_value !== intensity ||
    (intensity !== null && s.intensity_metric !== card.intensityMetric)
  )
}

// The reps a capped WOD's progress makes: what the capped score is, kept in the draft.
export function progressTotal(wod: WodCard): number {
  return progressReps(wod.lines, wod.rounds, wod.draft.progress)
}

// Whether a WOD takes its capped score as progress (full rounds, the next exercise, its reps)
// rather than as reps typed in.
export const takesProgress = (wod: WodCard): boolean => wod.scoreType === 'for_time' && progressEnabled(wod.lines)

// What the list of a WOD marks as done: the lines before the next one, and the reps of that one.
export function listMark(wod: WodCard): { step: number; reps: number } | null {
  if (!wod.draft.capped || !takesProgress(wod)) return null
  const p = clampProgress(wod.lines, wod.rounds, wod.draft.progress)
  return { step: p.step, reps: p.reps }
}

// What a WOD logs per movement besides its score, added up into it: the reps of an EMOM (e.g. the
// max reps of a 20 s station) or of a max reps WOD, the calories of a max calories one (e.g. each
// interval of an erg pyramid); null for the other formats.
export function partsUnit(wod: WodCard): 'reps' | 'cal' | null {
  if (wod.scoreType === 'emom' || wod.scoreType === 'max_reps') return 'reps'
  return wod.scoreType === 'max_calories' ? 'cal' : null
}

// The rounds the parts are entered in. An EMOM is done one movement per interval, so a round is
// one pass through its movements (an EMOM of 12:00 with 4 movements has 3 rounds), the last one
// short when the intervals don't divide evenly; other WODs go through their lines in their rounds.
export function partRoundCount(wod: WodCard): number {
  const n = wod.lines.length
  return wod.scoreType === 'emom' && wod.intervals && n ? Math.ceil(wod.intervals / n) : wod.rounds
}

// The movements (indexes into lines) done in a round, from 1.
export function partLinesIn(wod: WodCard, round: number): number[] {
  const n = wod.lines.length
  const done = wod.scoreType === 'emom' && wod.intervals ? wod.intervals - (round - 1) * n : n
  return wod.lines.map((_, i) => i).filter((i) => i < done)
}

// The value logged per round and movement ([round - 1][line]): its exercises' sets, the set number
// being the round; reps as the set's reps, calories as its load (in cal).
export function partValues(wod: WodCard, cards: ExerciseCard[]): (number | null)[][] {
  const cal = partsUnit(wod) === 'cal'
  return Array.from({ length: partRoundCount(wod) }, (_, r) =>
    wod.lines.map((line) => {
      const logged = cards.find((c) => c.key === line.key)?.rows.find((x) => x.number === r + 1)?.logged
      return (cal ? logged?.load_value : logged?.reps_value) ?? null
    }),
  )
}

// The values of all rounds together.
export const partsTotal = (grid: (number | null)[][]): number => grid.flat().reduce<number>((sum, v) => sum + (v ?? 0), 0)

// The total of the parts logged, null without any.
export function partsTotalOf(wod: WodCard, cards: ExerciseCard[]): number | null {
  const grid = partValues(wod, cards)
  return grid.flat().some((v) => v !== null) ? partsTotal(grid) : null
}

// A WOD with parts logged is scored by them: their total as its reps or calories, and an EMOM's
// rounds the intervals that have reps (an interval without any wasn't done). Without parts the
// score stays as entered by hand (an EMOM then has no total reps).
export function fillPartsScore(wod: WodCard, cards: ExerciseCard[]) {
  const grid = partValues(wod, cards)
  wod.partsTotal = partsTotalOf(wod, cards)
  if (wod.partsTotal === null) {
    if (wod.scoreType === 'emom') wod.draft.reps = null
    return
  }
  if (wod.scoreType === 'max_calories') wod.draft.calories = wod.partsTotal
  else wod.draft.reps = wod.partsTotal
  if (wod.scoreType === 'emom') wod.draft.rounds = grid.flatMap((r, n) => partLinesIn(wod, n + 1).filter((i) => (r[i] ?? 0) > 0)).length
}

// Gives each card the note on the exercise at its place in the session, if any.
export function attachNotes(cards: ExerciseCard[], notes: ExerciseNote[]) {
  for (const card of cards) {
    card.note = notes.find((n) => n.block_position === card.blockPosition && n.exercise_position === card.position && n.exercise_id === card.exerciseId) ?? null
  }
}
