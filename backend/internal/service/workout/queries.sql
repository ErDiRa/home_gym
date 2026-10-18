-- name: GetActiveTraining :one
-- The training the week is planned by: the user's active one (at most one).
SELECT id, name, start_date, end_date, sessions_per_week
FROM trainings
WHERE user_id = $1 AND active AND deleted_at IS NULL;

-- name: ListTrainingUnits :many
-- The units of a training, done in turn.
SELECT id, name FROM training_units
WHERE training_id = $1
ORDER BY position;

-- name: GetUnit :one
-- A unit of one of the user's trainings, for starting a workout of it.
SELECT tu.id, tu.name, tu.training_id
FROM training_units tu
JOIN trainings t ON t.id = tu.training_id AND t.user_id = $2 AND t.deleted_at IS NULL
WHERE tu.id = $1;

-- name: GetLastUnitID :one
-- The unit of the user's last finished workout of a training, to suggest the next one.
SELECT unit_id FROM workouts
WHERE user_id = $1 AND training_id = $2 AND unit_id IS NOT NULL
  AND completed_at IS NOT NULL AND deleted_at IS NULL
ORDER BY completed_at DESC
LIMIT 1;

-- name: CreateWorkout :one
-- Fails on idx_workouts_user_in_progress when the user has a workout in progress.
INSERT INTO workouts (user_id, training_id, unit_id, unit_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetWorkout :one
SELECT * FROM workouts
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: GetCurrentWorkout :one
-- The user's workout in progress (at most one).
SELECT * FROM workouts
WHERE user_id = $1 AND completed_at IS NULL AND deleted_at IS NULL;

-- name: FinishWorkout :one
-- Finishing again keeps the first completion time and only replaces the notes.
UPDATE workouts
SET completed_at = COALESCE(completed_at, now()), notes = $3
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWorkout :execrows
UPDATE workouts SET deleted_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListWorkouts :many
-- The user's finished workouts in [from, to), newest first, with how many sets they logged.
SELECT
    w.id,
    w.training_id,
    w.unit_id,
    w.unit_name,
    w.started_at,
    w.completed_at,
    w.notes,
    (SELECT count(*) FROM workout_sets ws WHERE ws.workout_id = w.id)::bigint AS set_count
FROM workouts w
WHERE w.user_id = $1 AND w.deleted_at IS NULL
  AND w.completed_at >= sqlc.arg(from_time) AND w.completed_at < sqlc.arg(to_time)
ORDER BY w.completed_at DESC;

-- name: ListWorkoutSets :many
-- A workout's sets in the order of the unit, with the names of the exercise and units.
SELECT
    ws.id,
    ws.exercise_id,
    e.name AS exercise_name,
    ws.block_position,
    ws.exercise_position,
    ws.set_number,
    ws.reps_value,
    ws.reps_unit,
    ru.name AS reps_unit_name,
    ws.per_side,
    ws.load_value,
    ws.load_unit,
    COALESCE(lu.name, '')::text AS load_unit_name,
    ws.completed_at
FROM workout_sets ws
JOIN exercises e ON e.id = ws.exercise_id
JOIN rep_units ru ON ru.id = ws.reps_unit
LEFT JOIN load_units lu ON lu.id = ws.load_unit
WHERE ws.workout_id = $1
ORDER BY ws.block_position, ws.exercise_position, ws.set_number;

-- name: UpsertWorkoutSet :one
-- Logs a set, or corrects it when it was logged before. Only one of the user's own live
-- exercises is inserted; any other inserts nothing (no row).
INSERT INTO workout_sets (
    workout_id, exercise_id, block_position, exercise_position, set_number,
    reps_value, reps_unit, per_side, load_value, load_unit
)
SELECT
    sqlc.arg(workout_id), e.id, sqlc.arg(block_position), sqlc.arg(exercise_position), sqlc.arg(set_number),
    sqlc.arg(reps_value), sqlc.arg(reps_unit), sqlc.arg(per_side), sqlc.narg(load_value), sqlc.narg(load_unit)
FROM exercises e
WHERE e.id = sqlc.arg(exercise_id) AND e.user_id = sqlc.arg(user_id) AND e.deleted_at IS NULL
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
DELETE FROM workout_sets
WHERE id = $1 AND workout_id = $2;

-- name: ListRepUnits :many
SELECT id, name, min_value, max_value FROM rep_units
WHERE deleted_at IS NULL;

-- name: ListLoadUnits :many
SELECT id, name, min_value, max_value FROM load_units
WHERE deleted_at IS NULL;
