// The API's types, generated from backend/api/openapi.yaml into openapi.d.ts (npm run gen:api);
// the descriptions are in the spec. This file names them for the app and adds what only the UI
// needs (labels). Never redefine a type the server sends or receives here: change the spec.
import type { components } from './openapi'

type Schemas = components['schemas']

// Whether an exercise mainly improves mobility or strength.
export type ExerciseType = Schemas['ExerciseType']

export const exerciseTypeLabels: Record<ExerciseType, string> = {
  mobility: 'Mobility',
  strength: 'Strength',
}

// An exercise as listed: its links by name (focuses with alias), for display and filtering.
export type Exercise = Schemas['Exercise']
// A training focus; the short alias is shown as a tag, the full name on hover.
export type Focus = Schemas['Focus']
// An entry of a reference list (joints, muscle groups) an exercise picks from.
export type Option = Schemas['Option']
// What the exercise form sends when creating or updating an exercise.
export type ExerciseInput = Schemas['ExerciseInput']
// One exercise as returned by the detail endpoint, with the ids the form needs.
export type ExerciseDetail = Schemas['ExerciseDetail']

// A training runs over whole weeks from a Monday. Its sessions (e.g. "Session 1") hold blocks
// (e.g. "Warm-up", "Superset"), and blocks hold one or more exercises with their prescription;
// its schedule plans each week with up to 7 of its sessions. Session templates of the same shape
// live in the library; adding one to a training copies it. The *Input types are what the forms
// send, the others what the server returns.

// The kind of a training block (single, superset, complex, WOD...) with its rules from the server,
// so new kinds need no code.
export type BlockKind = Schemas['BlockKind']
// How a WOD block is scored (see components/training/wod.ts).
export type ScoreType = Schemas['ScoreType']
export type Score = Schemas['Score']

// A unit planned reps (reps, s, m...) or load (kg, % 1RM, RPE...) are measured in. Only load
// units have kg_per_unit and percent_of_max.
export type Unit = Schemas['Unit'] & Partial<Pick<Schemas['LoadUnit'], 'kg_per_unit' | 'percent_of_max'>>

// Planned reps and load. A value is null when not set; a max turns it into a range (8-10).
export type Amount = Schemas['Amount']
// The unit names of an amount, as returned by the server.
export type AmountUnitNames = Schemas['AmountUnitNames']

export type TrainingExerciseInput = Schemas['TrainingExerciseInput']
// An exercise in a block as returned, with the exercise's name and type for display.
export type TrainingExercise = Schemas['TrainingExercise']
// A block's sets and amount are only used when its kind shares them; otherwise each exercise
// has its own. The sets of a WOD block are its rounds (null = not fixed).
export type TrainingBlockInput = Schemas['TrainingBlockInput']
export type TrainingSessionInput = Schemas['TrainingSessionInput']
// A planned session of a week: session is its index in TrainingInput.sessions (new sessions
// have no id yet).
export type SlotInput = Schemas['SlotInput']
// What the training form sends; every list is in order.
export type TrainingInput = Schemas['TrainingInput']
export type TrainingBlock = Schemas['TrainingBlock']
export type TrainingSession = Schemas['TrainingSession']
// A planned session of a week.
export type Slot = Schemas['Slot']
// A whole training as the edit form and the tracker load it.
export type TrainingDetail = Schemas['TrainingDetail']
// A row of the trainings list.
export type Training = Schemas['Training']

// A cycle's goal: what it should achieve, optionally a target to check at the end, and how it
// went. Goals are rewritten on save, in order; ids only key them in lists.
export type GoalStatus = Schemas['GoalStatus']

export const goalStatusLabels: Record<GoalStatus, string> = {
  open: 'Open',
  achieved: 'Achieved',
  missed: 'Missed',
}

export type CycleGoalInput = Schemas['CycleGoalInput']
export type CycleGoal = Schemas['CycleGoal']
// A cycle: a planned period of whole weeks from a Monday with goals, grouping trainings (each in
// at most one cycle; they may run outside its weeks).
export type CycleInput = Schemas['CycleInput']
// A training of a cycle, as the cycle lists it.
export type CycleTraining = Schemas['CycleTraining']
export type Cycle = Schemas['Cycle']
// A whole cycle with its trainings, imported at once (all or nothing).
export type CycleImportInput = Schemas['CycleImportInput']
// What a cycle import created: the cycle and its trainings, in the order of the file.
export type ImportedCycle = Schemas['ImportedCycle']
// A row of the cycles list.
export type CycleSummary = Schemas['CycleSummary']

// A session template of the library, as listed.
export type LibrarySession = Schemas['LibrarySession']
// A session template with its blocks; the form sends a TrainingSessionInput without id.
export type LibrarySessionDetail = Schemas['LibrarySessionDetail']

// An entry of the user's 1RM history of an exercise, always in a weight unit; the newest is the
// current max.
export type ExerciseMax = Schemas['ExerciseMax']
export type ExerciseMaxInput = Schemas['ExerciseMaxInput']
// How the user rated an exercise (1-5), with their notes.
export type ExerciseRating = Schemas['ExerciseRating']

// A performed set at its place in the session and what was actually done.
export type SetInput = Schemas['SetInput']
// A set as stored, which logging a set returns.
export type LoggedSet = Schemas['LoggedSet']
// A set as a workout lists it, with the names for display.
export type WorkoutSet = Schemas['WorkoutSet']
// A session done: started from a planned slot of a week, done on any day. completed_at is null
// while in progress.
export type Workout = Schemas['Workout']
// The score of a WOD block as done; score_type decides which values are set (see the spec).
export type ResultInput = Schemas['ResultInput']
export type WorkoutResult = Schemas['WorkoutResult']
// A finished workout as listed.
export type WorkoutSummary = Schemas['WorkoutSummary']
// An active training as the tracker offers it.
export type ActiveTraining = Schemas['ActiveTraining']
// The cycle of the training shown in the tracker, with the week of the cycle.
export type WeekCycle = Schemas['WeekCycle']
// A week as the tracker shows it: the active trainings to switch between, the sessions the
// chosen one plans for the week and the workouts finished.
export type Week = Schemas['Week']

// Whether WHOOP can be connected (the server has a WHOOP app configured) and whether it is.
export type WhoopStatus = Schemas['WhoopStatus']
// A WHOOP activity of the workout's day, to pick the one to link.
export type WhoopCandidate = Schemas['WhoopCandidate']
// The WHOOP activity linked to a workout, with the recovery of that day.
export type WhoopData = Schemas['WhoopData']
