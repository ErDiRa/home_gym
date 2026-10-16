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

// A training belongs to one client and is a tree: units (e.g. "Day A") hold blocks (e.g.
// "Warm-up", "Superset"), and blocks hold one or more exercises with their prescription.
// reps is free text, e.g. '8-10'.
export interface TrainingExerciseInput {
  exercise_id: string
  sets: number
  reps: string
  notes: string
}

export interface TrainingExercise extends TrainingExerciseInput {
  name: string
  type: ExerciseType
}

export interface TrainingBlockInput {
  name: string
  notes: string
  exercises: TrainingExerciseInput[]
}

// When a unit is done: recurring weekdays (ISO, 1 = Monday) and/or specific dates.
export interface UnitSchedule {
  weekdays: number[]
  dates: string[] // YYYY-MM-DD
}

export interface TrainingUnitInput extends UnitSchedule {
  // Set for existing units, so they keep their id.
  id?: string
  name: string
  notes: string
  blocks: TrainingBlockInput[]
}

// What the training form sends; every list is in order.
export interface TrainingInput {
  client_id: string
  name: string
  description: string
  active: boolean
  units: TrainingUnitInput[]
}

export interface TrainingBlock {
  id: string
  name: string
  notes: string
  exercises: TrainingExercise[]
}

export interface TrainingUnit extends UnitSchedule {
  id: string
  name: string
  notes: string
  blocks: TrainingBlock[]
}

export interface TrainingDetail {
  id: string
  client_id: string
  name: string
  description: string
  active: boolean
  units: TrainingUnit[]
}

// A row of the trainings list.
export interface Training {
  id: string
  name: string
  description: string
  active: boolean
  updated_at: string
  client_id: string
  client_first_name: string
  client_last_name: string
  unit_count: number
  exercise_count: number
}

// Clients are the user accounts admins manage.
export interface Client {
  id: string
  email: string
  first_name: string
  last_name: string
  created_at: string
  training_count: number
  active_training: string // name, empty if none
}

export interface ClientInput {
  email: string
  first_name: string
  last_name: string
}

// Only when creating a client; the password can't be changed afterwards.
export interface NewClientInput extends ClientInput {
  password: string
}

// A training as listed on its client's page.
export interface ClientTraining {
  id: string
  name: string
  active: boolean
  updated_at: string
  unit_count: number
}

export interface ClientDetail extends ClientInput {
  id: string
  created_at: string
  trainings: ClientTraining[]
}
