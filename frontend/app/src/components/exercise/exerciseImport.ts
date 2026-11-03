import { exerciseTypeLabels, type Exercise, type ExerciseInput, type ExerciseType, type Focus, type Option } from '../../types'
import { check, isObject, isOptionalText, isText, nameKey, parseJSON } from '../training/trainingImport'

// Exercises as JSON, e.g. collected elsewhere or written by hand. Joints, muscle groups and
// focuses are named instead of referenced by id, so the file can be written without the database;
// the import resolves the names and creates the exercises the user doesn't have yet.
export const EXERCISE_IMPORT_FORMAT = 'exercise-import/1'

// At most this many links per exercise, as the server keeps.
const MAX_VIDEOS = 10

export interface ImportExerciseDef {
  name: string
  type?: ExerciseType // strength when missing
  description?: string
  joints?: string[]
  muscle_groups?: string[]
  focuses?: string[] // by name or alias
  video_urls?: string[]
}

export interface ExerciseImport {
  format: typeof EXERCISE_IMPORT_FORMAT
  exercises: ImportExerciseDef[]
}

// Reads and checks an exercise import; throws an ImportError naming the problem.
export function parseExerciseImport(text: string): ExerciseImport {
  const data = parseJSON(text)
  check(data.format === EXERCISE_IMPORT_FORMAT, '', `Expected "format": "${EXERCISE_IMPORT_FORMAT}".`)
  check(Array.isArray(data.exercises) && data.exercises.length > 0, '', 'exercises must be a list with at least one exercise.')
  const seen = new Set<string>()
  ;(data.exercises as unknown[]).forEach((raw, i) => {
    const where = `Exercise ${i + 1}`
    check(isObject(raw), where, 'must be an object.')
    const e = raw as Record<string, unknown>
    check(isText(e.name), where, 'needs a name.')
    const name = `${where} "${(e.name as string).trim()}"`
    check(!seen.has(nameKey(e.name as string)), name, 'is in the file twice.')
    seen.add(nameKey(e.name as string))
    check(e.type === undefined || (typeof e.type === 'string' && e.type in exerciseTypeLabels), name, `type must be one of ${Object.keys(exerciseTypeLabels).join(', ')}.`)
    check(isOptionalText(e.description), name, 'description must be text.')
    for (const list of ['joints', 'muscle_groups', 'focuses', 'video_urls'] as const) {
      const v = e[list]
      check(v === undefined || (Array.isArray(v) && v.every((x) => typeof x === 'string')), name, `${list} must be a list of names.`)
    }
    const videos = (e.video_urls as string[] | undefined) ?? []
    check(videos.length <= MAX_VIDEOS, name, `has more than ${MAX_VIDEOS} video_urls.`)
    check(videos.every((u) => !u.trim() || /^https?:\/\//i.test(u.trim())), name, 'video_urls must be http(s) links.')
  })
  return data as unknown as ExerciseImport
}

// One exercise of the file as it will be imported: the input to create it, or none when the user
// has it already (by name); with the names nothing matched, which are left out.
export interface PlannedExercise {
  name: string
  type: ExerciseType
  input: ExerciseInput | null
  joints: string[]
  muscleGroups: string[]
  focuses: string[]
  unknown: string[]
}

export interface ExerciseLookups {
  joints: Option[]
  muscleGroups: Option[]
  focuses: Focus[]
}

// Resolves the file's names against the user's exercises and the lookup lists (trimmed, ignoring
// case; a focus also by its alias).
export function planExerciseImport(imp: ExerciseImport, existing: Exercise[], lookups: ExerciseLookups): PlannedExercise[] {
  const have = new Set(existing.map((e) => nameKey(e.name)))
  return imp.exercises.map((e) => {
    const unknown: string[] = []
    const resolve = <T extends { id: string; name: string }>(names: string[] | undefined, options: T[], matches: (o: T, key: string) => boolean) => {
      const found: T[] = []
      for (const n of names ?? []) {
        if (!n.trim()) continue
        const o = options.find((x) => matches(x, nameKey(n)))
        if (!o) unknown.push(n.trim())
        else if (!found.includes(o)) found.push(o)
      }
      return found
    }
    const byName = (o: { name: string }, key: string) => nameKey(o.name) === key
    const joints = resolve(e.joints, lookups.joints, byName)
    const muscleGroups = resolve(e.muscle_groups, lookups.muscleGroups, byName)
    const focuses = resolve(e.focuses, lookups.focuses, (o, key) => byName(o, key) || nameKey(o.alias) === key)
    const name = e.name.trim()
    const type = e.type ?? 'strength'
    return {
      name,
      type,
      input: have.has(nameKey(name))
        ? null
        : {
            name,
            type,
            description: e.description?.trim() ?? '',
            video_urls: (e.video_urls ?? []).map((u) => u.trim()).filter(Boolean),
            joint_ids: joints.map((o) => o.id),
            muscle_group_ids: muscleGroups.map((o) => o.id),
            focus_ids: focuses.map((o) => o.id),
          },
      joints: joints.map((o) => o.name),
      muscleGroups: muscleGroups.map((o) => o.name),
      focuses: focuses.map((o) => o.alias || o.name),
      unknown,
    }
  })
}
