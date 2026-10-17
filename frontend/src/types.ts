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
  video_urls: string[] // in display order
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
  video_urls: string[] // in display order
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
// The kind of a training block (single, superset, complex...). Its rules come from the server,
// so new kinds need no code: how many exercises it takes, and whether sets, reps and load are
// set once on the block for all exercises (shared) or per exercise.
export interface BlockKind {
  code: string
  name: string
  description: string
  min_exercises: number
  max_exercises: number | null // null = no limit
  shared_sets: boolean
  shared_reps: boolean
  shared_load: boolean
}

// A unit planned reps (reps, s, m...) or load (kg, % 1RM, RPE...) are measured in. min/max bound
// the values planned in it.
export interface Unit {
  id: string
  name: string
  min_value: number | null
  max_value: number | null
}

// Planned reps and load. A value is null when not set; a max turns it into a range (8-10).
export interface Amount {
  reps_value: number | null
  reps_max: number | null
  reps_unit: string | null
  per_side: boolean
  load_value: number | null
  load_max: number | null
  load_unit: string | null
}

// The unit names of an amount, as returned by the server.
export interface AmountUnitNames {
  reps_unit_name: string
  load_unit_name: string
}

export interface TrainingExerciseInput extends Amount {
  exercise_id: string
  sets: number | null // null when the block's kind shares sets
  tempo: string // free text, e.g. '31X1'
  notes: string
}

export interface TrainingExercise extends TrainingExerciseInput, AmountUnitNames {
  name: string
  type: ExerciseType
}

// A block's sets and amount are only used when its kind shares them; otherwise each exercise
// has its own.
export interface TrainingBlockInput extends Amount {
  kind: string
  name: string
  sets: number | null
  notes: string
  exercises: TrainingExerciseInput[]
}

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

export interface TrainingBlock extends Amount, AmountUnitNames {
  id: string
  kind: string
  name: string
  sets: number | null
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
