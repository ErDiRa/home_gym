// Whether an exercise mainly improves mobility or strength.
export type ExerciseType = 'mobility' | 'strength'

export const exerciseTypeLabels: Record<ExerciseType, string> = {
  mobility: 'Mobility',
  strength: 'Strength',
}

export interface Exercise {
  id: string
  name: string
  type: ExerciseType
  joints: string[]
  muscle_groups: string[]
  focuses: Focus[]
  description: string
  video_url: string
}

// A training focus; the short alias is shown as a tag, the full name on hover.
export interface Focus {
  id: string
  name: string
  alias: string
}

// An entry of a reference list (joints, muscle groups) an exercise picks from.
export interface Option {
  id: string
  name: string
}

// What the exercise form sends when creating or updating an exercise.
export interface ExerciseInput {
  name: string
  type: ExerciseType
  description: string
  video_url: string
  joint_ids: string[]
  muscle_group_ids: string[]
  focus_ids: string[]
}

// One exercise as returned by the detail endpoint, with the ids the form needs.
export interface ExerciseDetail extends ExerciseInput {
  id: string
}
