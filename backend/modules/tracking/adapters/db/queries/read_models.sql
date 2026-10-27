-- The read models: what the tracker and the dashboard show, mapped straight to the HTTP
-- responses (with the catalog's names, read through its contract).

-- name: GetWorkout :one
SELECT * FROM tracking.workouts
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListWorkouts :many
-- The user's finished workouts in [from, to), newest first, with how many sets they logged.
SELECT
    w.id,
    w.training_id,
    w.week_session_id,
    w.session_id,
    w.session_name,
    w.started_at,
    w.completed_at,
    w.notes,
    (SELECT count(*) FROM tracking.workout_sets ws WHERE ws.workout_id = w.id)::bigint AS set_count
FROM tracking.workouts w
WHERE w.user_id = $1 AND w.deleted_at IS NULL
  AND w.completed_at >= sqlc.arg(from_time) AND w.completed_at < sqlc.arg(to_time)
ORDER BY w.completed_at DESC;

-- name: ListWorkoutSets :many
-- A workout's sets in the order of the session.
SELECT * FROM tracking.workout_sets
WHERE workout_id = $1
ORDER BY block_position, exercise_position, set_number;

-- name: ListWorkoutResults :many
-- The scores of a workout's WOD blocks, in the order of the session.
SELECT * FROM tracking.workout_results
WHERE workout_id = $1
ORDER BY block_position;

-- name: ListExerciseMaxes :many
-- A user's 1RM history, newest first per exercise: the first entry of an exercise is its current
-- max. Entries of deleted exercises are left out by the caller (the catalog knows them).
SELECT id, exercise_id, load_value, load_unit, achieved_on, workout_set_id, notes, created_at
FROM tracking.exercise_maxes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY exercise_id, achieved_on DESC, created_at DESC;

-- name: GetExerciseMax :one
SELECT id, exercise_id, load_value, load_unit, achieved_on, workout_set_id, notes, created_at
FROM tracking.exercise_maxes
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: CountExercisesDone :many
-- The exercises a user has done in finished workouts, with in how many workouts.
SELECT ws.exercise_id, COUNT(DISTINCT w.id)::bigint AS times_performed
FROM tracking.workout_sets ws
JOIN tracking.workouts w ON w.id = ws.workout_id
WHERE w.user_id = $1 AND w.completed_at IS NOT NULL AND w.deleted_at IS NULL
GROUP BY ws.exercise_id;
