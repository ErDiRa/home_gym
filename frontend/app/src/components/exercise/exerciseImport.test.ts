import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import type { Exercise } from '../../types'
import { parseExerciseImport, planExerciseImport, type ExerciseLookups } from './exerciseImport'

const fixture = readFileSync(new URL('./__fixtures__/exercises.json', import.meta.url), 'utf8')

const lookups: ExerciseLookups = {
  joints: [
    { id: 'hip', name: 'Hip' },
    { id: 'shoulder', name: 'Shoulder' },
  ],
  muscleGroups: [{ id: 'core', name: 'Core' }],
  focuses: [
    { id: 'hipstab', name: 'Hip stability', alias: 'HIP' },
    { id: 'mob', name: 'Mobility', alias: 'MOB' },
  ],
}
const existing = [{ id: 'ski', name: 'ski erg' }] as Exercise[]

describe('the exercise import', () => {
  it('creates the exercises the user lacks, with the names resolved', () => {
    const plan = planExerciseImport(parseExerciseImport(fixture), existing, lookups)
    expect(plan.map((p) => p.name)).toEqual(['Copenhagen Plank', 'Ski Erg', '90/90 Hip Switch'])
    expect(plan[0].input).toEqual({
      name: 'Copenhagen Plank',
      type: 'strength',
      description: 'Side plank with the top leg on a bench; the adductors hold the hips up.',
      video_urls: ['https://www.youtube.com/watch?v=example'],
      joint_ids: ['hip'],
      muscle_group_ids: ['core'],
      focus_ids: ['hipstab'],
    })
    // Names nothing matches are left out and reported.
    expect(plan[0].unknown).toEqual(['Adductors'])
  })

  it('skips the exercises the user has (by name, ignoring case)', () => {
    const plan = planExerciseImport(parseExerciseImport(fixture), existing, lookups)
    expect(plan[1].input).toBeNull()
  })

  it('matches names ignoring case and a focus by its alias', () => {
    const plan = planExerciseImport(parseExerciseImport(fixture), [], lookups)
    expect(plan[2].input).toMatchObject({ type: 'mobility', joint_ids: ['hip'], focus_ids: ['mob'] })
    expect(plan[2].unknown).toEqual(['Ankle'])
  })

  it('defaults the type to strength', () => {
    const plan = planExerciseImport(parseExerciseImport('{"format":"exercise-import/1","exercises":[{"name":"Plank"}]}'), [], lookups)
    expect(plan[0].input).toMatchObject({ type: 'strength', description: '', video_urls: [], joint_ids: [] })
  })

  it.each([
    ['not JSON', '{', 'This is not valid JSON.'],
    ['another format', '{"format":"training-import/1","exercises":[]}', 'Expected "format": "exercise-import/1".'],
    ['no exercises', '{"format":"exercise-import/1","exercises":[]}', 'exercises must be a list with at least one exercise.'],
    ['no name', '{"format":"exercise-import/1","exercises":[{"type":"strength"}]}', 'Exercise 1: needs a name.'],
    ['a name twice', '{"format":"exercise-import/1","exercises":[{"name":"Plank"},{"name":" plank "}]}', 'Exercise 2 "plank": is in the file twice.'],
    ['an unknown type', '{"format":"exercise-import/1","exercises":[{"name":"Plank","type":"cardio"}]}', 'Exercise 1 "Plank": type must be one of mobility, strength, endurance.'],
    ['joints not a list', '{"format":"exercise-import/1","exercises":[{"name":"Plank","joints":"Hip"}]}', 'Exercise 1 "Plank": joints must be a list of names.'],
    ['a video not a link', '{"format":"exercise-import/1","exercises":[{"name":"Plank","video_urls":["youtube"]}]}', 'Exercise 1 "Plank": video_urls must be http(s) links.'],
  ])('rejects %s', (_, text, message) => {
    expect(() => parseExerciseImport(text)).toThrow(message)
  })
})
