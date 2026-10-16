export interface Exercise {
  id: string
  name: string
  joints: string[]
  muscle_groups: string[]
  description: string
  video_url: string
}

// An entry of a reference list (joints, muscle groups) an exercise picks from.
export interface Option {
  id: string
  name: string
}

// What the exercise form sends when creating or updating an exercise.
export interface ExerciseInput {
  name: string
  description: string
  video_url: string
  joint_ids: string[]
  muscle_group_ids: string[]
}

// One exercise as returned by the detail endpoint, with the ids the form needs.
export interface ExerciseDetail extends ExerciseInput {
  id: string
}
