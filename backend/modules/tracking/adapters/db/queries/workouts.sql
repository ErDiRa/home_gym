-- Writing workouts with their sets and WOD scores (WorkoutRepository).

-- name: CreateWorkout :one
-- Fails on idx_workouts_slot_in_progress when the slot already has a workout in progress.
INSERT INTO tracking.workouts (user_id, training_id, week_session_id, session_id, session_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetSlotWorkoutInProgress :one
-- The workout in progress of a slot of the user's (at most one).
SELECT id FROM tracking.workouts
WHERE week_session_id = $1 AND user_id = $2 AND completed_at IS NULL AND deleted_at IS NULL;

-- name: UserOwnsWorkout :one
SELECT EXISTS (SELECT 1 FROM tracking.workouts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL);

-- name: FinishWorkout :execrows
-- Finishing again keeps the first completion time and only replaces the notes, unless a
-- completion time is given (the workout was done on another day): then the start moves along,
-- keeping the duration.
UPDATE tracking.workouts
SET completed_at = COALESCE(sqlc.narg(completed_at)::timestamptz, completed_at, now()),
    started_at = started_at
        + (COALESCE(sqlc.narg(completed_at)::timestamptz, completed_at, now()) - COALESCE(completed_at, now())),
    notes = sqlc.arg(notes)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: DeleteWorkout :execrows
UPDATE tracking.workouts SET deleted_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: UpsertWorkoutSet :one
-- Logs a set, or corrects it when it was logged before. The exercise is checked to be one of the
-- user's live ones before (the catalog's contract).
INSERT INTO tracking.workout_sets (
    workout_id, exercise_id, block_position, exercise_position, set_number,
    reps_value, reps_unit, per_side, load_value, load_unit
) VALUES (
    sqlc.arg(workout_id), sqlc.arg(exercise_id), sqlc.arg(block_position), sqlc.arg(exercise_position), sqlc.arg(set_number),
    sqlc.narg(reps_value), sqlc.arg(reps_unit), sqlc.arg(per_side), sqlc.narg(load_value), sqlc.narg(load_unit)
)
ON CONFLICT (workout_id, block_position, exercise_position, set_number)
DO UPDATE SET exercise_id = EXCLUDED.exercise_id,
              reps_value = EXCLUDED.reps_value,
              reps_unit = EXCLUDED.reps_unit,
              per_side = EXCLUDED.per_side,
              load_value = EXCLUDED.load_value,
              load_unit = EXCLUDED.load_unit,
              completed_at = now()
RETURNING *;

-- name: DeleteWorkoutSet :execrows
DELETE FROM tracking.workout_sets
WHERE id = $1 AND workout_id = $2;

-- name: UpsertWorkoutResult :one
-- Logs the score of a block, or corrects it when it was logged before.
INSERT INTO tracking.workout_results (
    workout_id, block_position, score_type, time_seconds, rounds, reps, calories, rx, notes
) VALUES (
    sqlc.arg(workout_id), sqlc.arg(block_position), sqlc.arg(score_type), sqlc.narg(time_seconds),
    sqlc.narg(rounds), sqlc.narg(reps), sqlc.narg(calories), sqlc.arg(rx), sqlc.arg(notes)
)
ON CONFLICT (workout_id, block_position)
DO UPDATE SET score_type = EXCLUDED.score_type,
              time_seconds = EXCLUDED.time_seconds,
              rounds = EXCLUDED.rounds,
              reps = EXCLUDED.reps,
              calories = EXCLUDED.calories,
              rx = EXCLUDED.rx,
              notes = EXCLUDED.notes,
              completed_at = now()
RETURNING *;

-- name: DeleteWorkoutResult :execrows
DELETE FROM tracking.workout_results
WHERE id = $1 AND workout_id = $2;
