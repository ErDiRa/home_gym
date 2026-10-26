import { describe, expect, it } from 'vitest'
import { searchExercises } from './components/exercise/exerciseSearch'
import { searchTrainings } from './components/training/trainingSearch'
import { fuzzySearch } from './fuzzySearch'
import type { Exercise, Training } from './types'

const exercise = (name: string, more: Partial<Exercise> = {}): Exercise => ({
  id: name,
  name,
  type: 'strength',
  joints: [],
  muscle_groups: [],
  focuses: [],
  description: '',
  video_urls: [],
  ...more,
})

const exercises = [
  exercise('Back Squat', { joints: ['Knee', 'Hip'], muscle_groups: ['Quadriceps'] }),
  exercise('Push-up', { joints: ['Shoulder'], muscle_groups: ['Chest'] }),
  exercise('Bulgarian Split Squat', { joints: ['Knee'], muscle_groups: ['Quadriceps'] }),
  exercise('Knee Push-up', { joints: ['Knee', 'Shoulder'], muscle_groups: ['Chest'] }),
  exercise('Nordic Curl', {
    muscle_groups: ['Hamstrings'],
    focuses: [{ id: 'f1', name: 'Knee health', alias: 'KH' }],
  }),
  exercise('Überzug', { muscle_groups: ['Lats'] }),
]
const names = (query: string) => searchExercises(exercises, query).map((e) => e.name)

describe('searchExercises', () => {
  it('returns everything in order without a query or with one letter', () => {
    expect(searchExercises(exercises, '')).toBe(exercises)
    expect(searchExercises(exercises, '  k ')).toBe(exercises)
  })

  it('allows a typo', () => {
    expect(names('sqaut')).toEqual(expect.arrayContaining(['Back Squat', 'Bulgarian Split Squat']))
    expect(names('hamstirngs')).toEqual(['Nordic Curl'])
  })

  it('finds by joint, muscle group and focus alias', () => {
    expect(names('shoulder')).toEqual(expect.arrayContaining(['Push-up', 'Knee Push-up']))
    expect(names('chest')).toHaveLength(2)
    expect(names('kh')).toEqual(['Nordic Curl'])
  })

  it('needs every word to match, in any field', () => {
    expect(names('knee chest')).toEqual(['Knee Push-up'])
    expect(names('quadriceps split')).toEqual(['Bulgarian Split Squat'])
  })

  it('ranks a match in the name first', () => {
    expect(names('knee')[0]).toBe('Knee Push-up')
  })

  it('ignores accents', () => {
    expect(names('uberzug')).toEqual(['Überzug'])
  })

  it('rebuilds the index for a new list', () => {
    expect(searchExercises([exercise('Deadlift')], 'deadlift')).toHaveLength(1)
    expect(names('deadlift')).toEqual([])
  })
})

describe('searchTrainings', () => {
  const training = (name: string, active: boolean, cycle_name: string | null = null): Training => ({
    id: name,
    cycle_id: cycle_name,
    cycle_name,
    name,
    description: '',
    active,
    start_date: '2026-10-05',
    end_date: '2026-11-29',
    weeks: 8,
    updated_at: '2026-10-01T00:00:00Z',
    session_count: 3,
    planned_count: 24,
  })
  const trainings = [training('Strength', true, 'Master Diesel'), training('Mobility', false), training('Warm-ups', true)]

  it('finds "active" ones only', () => {
    expect(searchTrainings(trainings, 'active').map((t) => t.name)).toEqual(['Strength', 'Warm-ups'])
  })

  it('finds by cycle name with a typo', () => {
    expect(searchTrainings(trainings, 'deisel').map((t) => t.name)).toEqual(['Strength'])
  })
})

describe('fuzzySearch', () => {
  it('does not match far-off words', () => {
    const search = fuzzySearch<{ text: string }>([{ name: 'text', weight: 1, get: (o) => o.text }])
    const words = ['pull', 'push', 'press', 'plank'].map((text) => ({ text }))
    expect(search(words, 'push').map((o) => o.text)).toEqual(['push'])
    expect(search(words, 'press').map((o) => o.text)).toEqual(['press'])
  })
})
