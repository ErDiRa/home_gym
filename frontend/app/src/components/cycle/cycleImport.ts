import type { CycleImportInput, Exercise, GoalStatus } from '../../types'
import { addDays } from '../../dates'
import { sessionProblem, toInput } from '../training/trainingForm'
import {
  check,
  checkNewExercises,
  checkSessions,
  checkTrainingImport,
  IMPORT_FORMAT,
  ImportError,
  isMondayDate,
  isObject,
  isOptionalText,
  isText,
  nameKey,
  parseJSON,
  toImportedTraining,
  type ImportNewExercise,
  type ImportSession,
  type TrainingImport,
} from '../training/trainingImport'

// A whole cycle as JSON: its period and goals, and its trainings, each placed by the cycle week it
// starts in (so moving the cycle is one date). The sessions are those of a training import.
export const CYCLE_IMPORT_FORMAT = 'cycle-import/1'

const MAX_GOALS = 20
const MAX_TRAININGS = 60
const MAX_WEEKS = 52
const GOAL_STATUSES: GoalStatus[] = ['open', 'achieved', 'missed']

export interface CycleImportGoal {
  title: string
  target?: string
  status?: GoalStatus
  notes?: string
}

export interface CycleImportTraining {
  week: number // the cycle week it starts in, from 1
  weeks?: number // how many weeks it runs (default 1), its sessions planned in each
  name: string
  description?: string
  active?: boolean // default true
  sessions: ImportSession[]
  new_exercises?: ImportNewExercise[]
}

export interface CycleImport {
  format: typeof CYCLE_IMPORT_FORMAT
  name: string
  description?: string
  start_date: string // a Monday: the cycle's first week
  weeks: number
  goals?: CycleImportGoal[]
  trainings?: CycleImportTraining[]
  new_exercises?: ImportNewExercise[]
}

// An import file of either format.
export type AnyImport = { kind: 'training'; imp: TrainingImport } | { kind: 'cycle'; imp: CycleImport }

// Reads an import file, a single training or a whole cycle, telling them apart by format.
export function parseAnyImport(text: string, kinds: string[]): AnyImport {
  const data = parseJSON(text)
  if (data.format === IMPORT_FORMAT) return { kind: 'training', imp: checkTrainingImport(data, kinds) }
  if (data.format === CYCLE_IMPORT_FORMAT) return { kind: 'cycle', imp: checkCycleImport(data, kinds) }
  throw new ImportError(
    `Expected "format": "${IMPORT_FORMAT}" (one training) or "${CYCLE_IMPORT_FORMAT}" (a whole cycle).`,
  )
}

// Reads and checks a cycle import against the block kinds there are; the values of its trainings
// are checked later like a training import's (the form's checks).
export function parseCycleImport(text: string, kinds: string[]): CycleImport {
  return checkCycleImport(parseJSON(text), kinds)
}

function checkCycleImport(c: Record<string, unknown>, kinds: string[]): CycleImport {
  check(c.format === CYCLE_IMPORT_FORMAT, '', `Expected "format": "${CYCLE_IMPORT_FORMAT}".`)
  check(isText(c.name) && c.name.trim().length <= 100, '', 'The cycle needs a name (max 100 characters).')
  check(isOptionalText(c.description), '', 'description must be text.')
  check(isMondayDate(c.start_date), '', 'start_date must be a Monday as YYYY-MM-DD.')
  const weeks = c.weeks
  check(Number.isInteger(weeks) && (weeks as number) >= 1 && (weeks as number) <= MAX_WEEKS, '', `weeks must be 1-${MAX_WEEKS}.`)

  if (c.goals !== undefined) {
    check(Array.isArray(c.goals), '', 'goals must be a list.')
    check(c.goals.length <= MAX_GOALS, '', `A cycle has at most ${MAX_GOALS} goals.`)
    c.goals.forEach((g, i) => {
      const where = `Goal ${i + 1}`
      check(isObject(g), where, 'must be an object.')
      check(isText(g.title) && g.title.trim().length <= 200, where, 'needs a title (max 200 characters).')
      check(isOptionalText(g.target) && isOptionalText(g.notes), where, 'target and notes must be text.')
      check(g.status === undefined || GOAL_STATUSES.includes(g.status as GoalStatus), where, 'status must be open, achieved or missed.')
    })
  }

  if (c.trainings !== undefined) {
    check(Array.isArray(c.trainings), '', 'trainings must be a list.')
    check(c.trainings.length <= MAX_TRAININGS, '', `A cycle import holds at most ${MAX_TRAININGS} trainings.`)
    c.trainings.forEach((t, i) => checkCycleTraining(t, i, weeks as number, kinds))
  }
  checkNewExercises(c.new_exercises, '')
  return c as unknown as CycleImport
}

function checkCycleTraining(t: unknown, i: number, cycleWeeks: number, kinds: string[]) {
  const position = `Training ${i + 1}`
  check(isObject(t), position, 'must be an object.')
  check(
    t.start_date === undefined && t.format === undefined,
    position,
    "set the start on the cycle (start_date) and the training's week (week).",
  )
  check(isText(t.name), position, 'needs a name.')
  const where = `${position} ("${t.name.trim()}")`
  const week = t.week
  check(Number.isInteger(week) && (week as number) >= 1 && (week as number) <= cycleWeeks, where, `week must be 1-${cycleWeeks} (the cycle's weeks).`)
  const weeks = t.weeks ?? 1
  check(Number.isInteger(weeks) && (weeks as number) >= 1, where, 'weeks must be a whole number of at least 1.')
  check(
    (week as number) + (weeks as number) - 1 <= cycleWeeks,
    where,
    `${weeks} weeks from week ${week} run past the cycle's ${cycleWeeks} weeks.`,
  )
  check(t.active === undefined || typeof t.active === 'boolean', where, 'active must be true or false.')
  check(isOptionalText(t.description), where, 'description must be text.')
  checkSessions(t.sessions, kinds, where)
  checkNewExercises(t.new_exercises, where)
}

// The Monday a cycle's training starts on: its week of the cycle.
export const trainingStart = (cycleStart: string, week: number) => addDays(cycleStart, (week - 1) * 7)

// The suggestions for new exercises of the whole file, the cycle's and its trainings', each name
// once (trimmed, ignoring case); the first one wins.
export function mergedNewExercises(c: CycleImport): ImportNewExercise[] {
  const byName = new Map<string, ImportNewExercise>()
  for (const n of [...(c.new_exercises ?? []), ...(c.trainings ?? []).flatMap((t) => t.new_exercises ?? [])]) {
    if (!byName.has(nameKey(n.name))) byName.set(nameKey(n.name), n)
  }
  return [...byName.values()]
}

// The request that imports the cycle once every name is known (exerciseByName by nameKey), or the
// first problem the form's checks find in its trainings, naming the training.
export function toCycleImportInput(
  c: CycleImport,
  exerciseByName: Map<string, Exercise>,
): { input: CycleImportInput; problem: '' } | { input: null; problem: string } {
  const trainings = []
  for (const t of c.trainings ?? []) {
    const weeks = t.weeks ?? 1
    const start = trainingStart(c.start_date, t.week)
    const rows = toImportedTraining({ name: t.name, description: t.description, start_date: start, sessions: t.sessions }, exerciseByName, weeks)
    for (const s of rows.sessions) {
      const problem = sessionProblem(s)
      if (problem) return { input: null, problem: `Week ${t.week} "${rows.name}": ${problem}` }
    }
    trainings.push(
      toInput(
        { cycle_id: null, name: rows.name, description: rows.description, active: t.active ?? true, start_date: start, weeks },
        rows.sessions,
        rows.weeks,
      ),
    )
  }
  const goals = (c.goals ?? []).map((g) => ({
    title: g.title.trim(),
    target: g.target?.trim() ?? '',
    status: g.status ?? ('open' as const),
    notes: g.notes?.trim() ?? '',
  }))
  const cycle = { name: c.name.trim(), description: c.description?.trim() ?? '', start_date: c.start_date, weeks: c.weeks, goals }
  return { input: { cycle, trainings }, problem: '' }
}
