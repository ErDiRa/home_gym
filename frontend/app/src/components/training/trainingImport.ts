import type { Amount, Exercise, ExerciseType, ScoreType } from '../../types'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import { isMonday } from '../../dates'
import {
  DEFAULT_SETS,
  MAX_SLOTS,
  emptyAmount,
  emptyScore,
  newWeek,
  nextKey,
  blockLetter,
  type BlockRow,
  type SessionRow,
  type WeekRow,
} from './trainingForm'
import { scoreFormats } from './wod'

// A training of one week as JSON, e.g. converted from a plan written elsewhere. It
// names exercises, units, joints and muscle groups instead of using ids, so it can be written
// without the database; the import resolves the names and asks about the ones it doesn't know.
export const IMPORT_FORMAT = 'training-import/1'

export interface ImportAmount {
  value: number
  max?: number | null // the upper end of a range
  unit: string // a rep or load unit by name, e.g. "reps", "cal", "kg", "% 1RM"
  per_side?: boolean
}

export interface ImportExercise {
  exercise: string // by name
  sets?: number // when the block's kind doesn't share sets
  reps?: ImportAmount
  load?: ImportAmount
  tempo?: string
  notes?: string
}

export interface ImportBlock {
  kind: string // a block kind's code: single, superset, complex, wod
  name?: string
  notes?: string
  sets?: number // of a kind sharing sets; a WOD's rounds (optional)
  reps?: ImportAmount // when the kind shares them
  load?: ImportAmount
  score?: { type: ScoreType; time_seconds?: number | null; interval_seconds?: number | null } // a WOD's
  exercises: ImportExercise[]
}

export interface ImportSession {
  name: string
  notes?: string
  blocks: ImportBlock[]
}

// Suggestions for exercises the user doesn't have yet, prefilled when they are created.
export interface ImportNewExercise {
  name: string
  type?: ExerciseType
  joints?: string[]
  muscle_groups?: string[]
  video_urls?: string[]
}

export interface TrainingImport {
  format: typeof IMPORT_FORMAT
  cycle?: string // the name of the cycle it goes into, preselected in the form when it exists
  name: string
  description?: string
  start_date: string // a Monday, YYYY-MM-DD
  sessions: ImportSession[] // planned in this order in the training's one week
  new_exercises?: ImportNewExercise[]
}

// What the form takes over from an import.
export interface ImportedTraining {
  cycle?: string
  name: string
  description: string
  startDate: string
  sessions: SessionRow[]
  weeks: WeekRow[]
}

// Anything holding sessions to import: a training import or a training of a cycle import.
export type WithSessions = Pick<TrainingImport, 'sessions'>

// Names are matched trimmed and ignoring case.
export const nameKey = (name: string) => name.trim().toLowerCase()

// A problem of an import file, with a message for the user.
export class ImportError extends Error {}

export const isObject = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v)
export const isText = (v: unknown): v is string => typeof v === 'string' && v.trim() !== ''
export const isOptionalText = (v: unknown) => v === undefined || typeof v === 'string'
const isCount = (v: unknown) => Number.isInteger(v) && (v as number) >= 1 && (v as number) <= 100
const isSeconds = (v: unknown) => v === undefined || v === null || (Number.isInteger(v) && (v as number) >= 1)
export const isMondayDate = (v: unknown): v is string => typeof v === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(v) && isMonday(v)

export function check(ok: boolean, where: string, message: string): asserts ok {
  if (!ok) throw new ImportError(where ? `${where}: ${message}` : message)
}

// The JSON object of an import file.
export function parseJSON(text: string): Record<string, unknown> {
  let data: unknown
  try {
    data = JSON.parse(text)
  } catch {
    throw new ImportError('This is not valid JSON.')
  }
  check(isObject(data), '', 'Expected a JSON object.')
  return data
}

function checkAmount(v: unknown, where: string) {
  if (v === undefined || v === null) return
  check(isObject(v), where, 'must be an object with value and unit.')
  const a = v as Record<string, unknown>
  check(typeof a.value === 'number' && a.value >= 0, where, 'value must be a number of at least 0.')
  check(a.max === undefined || a.max === null || typeof a.max === 'number', where, 'max must be a number.')
  check(isText(a.unit), where, 'unit is missing.')
  check(a.per_side === undefined || typeof a.per_side === 'boolean', where, 'per_side must be true or false.')
}

// Reads and checks the JSON of a training import against the block kinds there are. The values
// are checked by the form afterwards (sets, units, scores), like anything entered there.
export function parseImport(text: string, kinds: string[]): TrainingImport {
  return checkTrainingImport(parseJSON(text), kinds)
}

// checkTrainingImport is parseImport for an already parsed file.
export function checkTrainingImport(t: Record<string, unknown>, kinds: string[]): TrainingImport {
  check(t.format === IMPORT_FORMAT, '', `Expected "format": "${IMPORT_FORMAT}".`)
  check(t.cycle === undefined || isText(t.cycle), '', 'cycle must be the name of a cycle.')
  check(isText(t.name), '', 'The training needs a name.')
  check(isOptionalText(t.description), '', 'description must be text.')
  check(isMondayDate(t.start_date), '', 'start_date must be a Monday as YYYY-MM-DD.')
  checkSessions(t.sessions, kinds, '')
  checkNewExercises(t.new_exercises, '')
  return t as unknown as TrainingImport
}

// Checks the sessions of a training (1 to 7, each with blocks and exercises); where prefixes the
// messages, e.g. 'Training 3 ("Week 3")'.
export function checkSessions(raw: unknown, kinds: string[], where: string) {
  const at = (s: string) => (where ? `${where}, ${s}` : s)
  check(Array.isArray(raw) && raw.length >= 1, where, 'There are no sessions.')
  check(raw.length <= MAX_SLOTS, where, `A week holds at most ${MAX_SLOTS} sessions.`)

  raw.forEach((rawSession, i) => {
    const sessionAt = at(`Session ${i + 1}`)
    check(isObject(rawSession), sessionAt, 'must be an object.')
    const s = rawSession
    check(isText(s.name), sessionAt, 'needs a name.')
    check(isOptionalText(s.notes), sessionAt, 'notes must be text.')
    check(Array.isArray(s.blocks) && s.blocks.length >= 1, sessionAt, 'has no blocks.')
    ;(s.blocks as unknown[]).forEach((rawBlock, j) => {
      const blockAt = at(`${s.name}, block ${blockLetter(j)}`)
      check(isObject(rawBlock), blockAt, 'must be an object.')
      const b = rawBlock
      check(typeof b.kind === 'string' && kinds.includes(b.kind), blockAt, `unknown kind "${String(b.kind)}" (${kinds.join(', ')}).`)
      check(isOptionalText(b.name), blockAt, 'name must be text.')
      check(isOptionalText(b.notes), blockAt, 'notes must be text.')
      check(b.sets === undefined || b.sets === null || isCount(b.sets), blockAt, 'sets must be 1-100.')
      checkAmount(b.reps, `${blockAt}, reps`)
      checkAmount(b.load, `${blockAt}, load`)
      if (b.score !== undefined && b.score !== null) {
        check(isObject(b.score), blockAt, 'score must be an object.')
        const score = b.score
        check(
          scoreFormats.some((f) => f.code === score.type),
          blockAt,
          `unknown score type "${String(score.type)}" (${scoreFormats.map((f) => f.code).join(', ')}).`,
        )
        check(isSeconds(score.time_seconds) && isSeconds(score.interval_seconds), blockAt, 'times are whole seconds.')
      }
      check(Array.isArray(b.exercises) && b.exercises.length >= 1, blockAt, 'has no exercises.')
      ;(b.exercises as unknown[]).forEach((rawExercise, k) => {
        const exerciseAt = `${blockAt}, exercise ${k + 1}`
        check(isObject(rawExercise), exerciseAt, 'must be an object.')
        const e = rawExercise
        check(isText(e.exercise), exerciseAt, 'needs the exercise name.')
        check(e.sets === undefined || e.sets === null || isCount(e.sets), `${blockAt}, ${e.exercise}`, 'sets must be 1-100.')
        checkAmount(e.reps, `${blockAt}, ${e.exercise}, reps`)
        checkAmount(e.load, `${blockAt}, ${e.exercise}, load`)
        check(isOptionalText(e.tempo), exerciseAt, 'tempo must be text.')
        check(isOptionalText(e.notes), exerciseAt, 'notes must be text.')
      })
    })
  })
}

// Checks a list of suggestions for new exercises, if there is one.
export function checkNewExercises(raw: unknown, where: string) {
  if (raw === undefined) return
  check(Array.isArray(raw), where, 'new_exercises must be a list.')
  raw.forEach((n, i) => {
    check(isObject(n) && isText(n.name), where ? `${where}, new_exercises ${i + 1}` : `new_exercises ${i + 1}`, 'needs a name.')
  })
}

// The import's error message, or a generic one for anything unexpected.
export const importErrorMessage = (e: unknown) => (e instanceof ImportError ? e.message : 'The import could not be read.')

// Every amount of the trainings with its part, block and exercise amounts alike.
function* amounts(trainings: WithSessions[]): Generator<[AmountPart, ImportAmount]> {
  for (const s of trainings.flatMap((t) => t.sessions))
    for (const b of s.blocks)
      for (const owner of [b, ...b.exercises]) {
        if (owner.reps) yield ['reps', owner.reps]
        if (owner.load) yield ['load', owner.load]
      }
}

// The exercise and unit names of the trainings (one import, or all of a cycle's) nobody has yet,
// each once, as first written.
export function unresolvedNames(imports: WithSessions | WithSessions[], exercises: Exercise[]) {
  const trainings = Array.isArray(imports) ? imports : [imports]
  const known = new Set(exercises.map((e) => nameKey(e.name)))
  const missing = new Map<string, string>()
  for (const s of trainings.flatMap((t) => t.sessions))
    for (const b of s.blocks)
      for (const e of b.exercises) if (!known.has(nameKey(e.exercise))) missing.set(nameKey(e.exercise), e.exercise.trim())

  const units = new Map<string, { part: AmountPart; name: string }>()
  for (const [part, a] of amounts(trainings)) {
    if (!unitIdOf(part, a.unit)) units.set(`${part}:${nameKey(a.unit)}`, { part, name: a.unit.trim() })
  }
  return { exercises: [...missing.values()], units: [...units.values()] }
}

const unitIdOf = (part: AmountPart, name: string) =>
  useLookupsStore()
    .unitsOf(part)
    .find((u) => nameKey(u.name) === nameKey(name))?.id ?? null

function toAmount(reps?: ImportAmount, load?: ImportAmount): Amount {
  const a = emptyAmount()
  if (reps) {
    a.reps_value = reps.value
    a.reps_max = reps.max ?? null
    a.reps_unit = unitIdOf('reps', reps.unit)
    a.per_side = reps.per_side ?? false
  }
  if (load) {
    a.load_value = load.value
    a.load_max = load.max ?? null
    a.load_unit = unitIdOf('load', load.unit)
  }
  return a
}

// Turns an import into the form's rows once every name is known (exerciseByName by nameKey):
// one session per imported session, all planned in order in each of the weeks (one for a training
// import; a cycle's training may run several). Sets are filled on both sides like for a loaded
// training, so the form's checks see what the kind uses.
export function toImportedTraining(
  imp: Pick<TrainingImport, 'cycle' | 'name' | 'description' | 'start_date' | 'sessions'>,
  exerciseByName: Map<string, Exercise>,
  weekCount = 1,
): ImportedTraining {
  const lookups = useLookupsStore()
  const sessions: SessionRow[] = imp.sessions.map((s) => ({
    key: nextKey(),
    name: s.name.trim(),
    notes: s.notes ?? '',
    blocks: s.blocks.map((b): BlockRow => {
      const scored = lookups.kindOf(b.kind)?.scored ?? false
      const blockSets = b.sets ?? b.exercises.find((e) => e.sets)?.sets ?? DEFAULT_SETS
      return {
        key: nextKey(),
        kind: b.kind,
        name: b.name ?? '',
        sets: blockSets,
        rounds: scored ? (b.sets ?? null) : null,
        notes: b.notes ?? '',
        ...toAmount(b.reps, b.load),
        ...(scored && b.score
          ? {
              score_type: b.score.type,
              time_seconds: b.score.time_seconds ?? null,
              interval_seconds: b.score.interval_seconds ?? null,
            }
          : emptyScore()),
        exercises: b.exercises.map((e) => {
          const exercise = exerciseByName.get(nameKey(e.exercise))!
          return {
            key: nextKey(),
            exercise_id: exercise.id,
            name: exercise.name,
            type: exercise.type,
            sets: e.sets ?? blockSets,
            tempo: e.tempo ?? '',
            notes: e.notes ?? '',
            ...toAmount(e.reps, e.load),
          }
        }),
      }
    }),
  }))
  return {
    cycle: imp.cycle?.trim(),
    name: imp.name.trim(),
    description: imp.description ?? '',
    startDate: imp.start_date,
    sessions,
    weeks: Array.from({ length: weekCount }, () => newWeek(sessions.map((s) => ({ key: nextKey(), sessionKey: s.key })))),
  }
}
