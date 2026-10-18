// Whether an exercise mainly improves mobility or strength.
export type ExerciseType = 'mobility' | 'strength'

export const exerciseTypeLabels: Record<ExerciseType, string> = {
  mobility: 'Mobility',
  strength: 'Strength',
}

// An exercise as listed: its links by name (focuses with alias), for display and filtering.
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

// A training is a tree: units (e.g. "Day A", done in turn, as many sessions a week as the
// training asks for) hold blocks (e.g. "Warm-up", "Superset"), and blocks hold one or more
// exercises with their prescription. The *Input types are what the form sends, the others
// what the server returns.

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

// An exercise in a block as returned, with the exercise's name and type for display.
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

export interface TrainingUnitInput {
  // Set for existing units, so they keep their id.
  id?: string
  name: string
  notes: string
  blocks: TrainingBlockInput[]
}

// What the training form sends; every list is in order.
export interface TrainingInput {
  name: string
  description: string
  active: boolean
  start_date: string // YYYY-MM-DD, first day of the period
  end_date: string // YYYY-MM-DD, last day of the period
  sessions_per_week: number // 1-14, done on any days
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

export interface TrainingUnit {
  id: string
  name: string
  notes: string
  blocks: TrainingBlock[]
}

// A whole training as the edit form and the tracker load it.
export interface TrainingDetail {
  id: string
  name: string
  description: string
  active: boolean
  start_date: string
  end_date: string
  sessions_per_week: number
  units: TrainingUnit[]
}

// A row of the trainings list.
export interface Training {
  id: string
  name: string
  description: string
  active: boolean
  start_date: string
  end_date: string
  sessions_per_week: number
  updated_at: string
  unit_count: number
  exercise_count: number
}

// How the user rated an exercise (1-5), with their notes.
export interface ExerciseRating {
  exercise_id: string
  rating: number
  notes: string
  updated_at: string
}

// A performed set: the exercise's place in the unit (block and position in the block, from 1),
// the set number and what was actually done.
export interface SetInput {
  exercise_id: string
  block_position: number
  exercise_position: number
  set_number: number
  reps_value: number
  reps_unit: string
  per_side: boolean
  load_value: number | null // null without load, e.g. bodyweight
  load_unit: string | null
}

// A set as stored, which logging a set returns.
export interface LoggedSet extends SetInput {
  id: string
  completed_at: string
}

// A set as a workout lists it, with the names for display.
export interface WorkoutSet extends LoggedSet {
  exercise_name: string
  reps_unit_name: string
  load_unit_name: string
}

// One session: a unit of a training, done on any day. completed_at is null while in progress.
// training_id and unit_id become null when the training or unit is deleted later.
export interface Workout {
  id: string
  training_id: string | null
  unit_id: string | null
  unit_name: string
  started_at: string
  completed_at: string | null
  notes: string
  sets: WorkoutSet[]
}

// A finished workout as listed.
export interface WorkoutSummary {
  id: string
  training_id: string | null
  unit_id: string | null
  unit_name: string
  started_at: string
  completed_at: string
  notes: string
  set_count: number
}

// A week as the tracker shows it: what the active training asks for, the workouts finished,
// the unit to do next and the workout in progress.
export interface Week {
  training: {
    id: string
    name: string
    start_date: string
    end_date: string
    sessions_per_week: number
  } | null // null when no training is active
  units: { id: string; name: string }[] // the active training's, in order
  done: WorkoutSummary[]
  next_unit_id: string | null
  current: Workout | null
}
