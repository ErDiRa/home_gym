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

-- name: ListExerciseNotes :many
-- The notes on a workout's exercises, in the order of the session.
SELECT * FROM tracking.workout_exercise_notes
WHERE workout_id = $1
ORDER BY block_position, exercise_position;

-- name: ListWorkoutResults :many
-- The scores of a workout's WOD blocks, in the order of the session.
SELECT * FROM tracking.workout_results
WHERE workout_id = $1
ORDER BY block_position;

-- name: ListExerciseMaxes :many
-- A user's 1RM history, newest first per exercise: the first entry of an exercise is its current
-- max. Entries of deleted exercises are left out by the caller (the catalog knows them).
SELECT id, exercise_id, reps, load_value, load_unit, achieved_on, workout_set_id, notes, created_at
FROM tracking.exercise_maxes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY exercise_id, achieved_on DESC, created_at DESC;

-- name: GetExerciseMax :one
SELECT id, exercise_id, reps, load_value, load_unit, achieved_on, workout_set_id, notes, created_at
FROM tracking.exercise_maxes
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: CountExercisesDone :many
-- The exercises a user has done in finished workouts, with in how many workouts.
SELECT ws.exercise_id, COUNT(DISTINCT w.id)::bigint AS times_performed
FROM tracking.workout_sets ws
JOIN tracking.workouts w ON w.id = ws.workout_id
WHERE w.user_id = $1 AND w.completed_at IS NOT NULL AND w.deleted_at IS NULL
GROUP BY ws.exercise_id;

-- name: ListLastLoads :many
-- The load a user last logged per exercise: the last set with a load (in session order) of the
-- newest finished workout that has one. Weight added to bodyweight isn't such a load.
SELECT DISTINCT ON (ws.exercise_id) ws.exercise_id, ws.load_value, ws.load_unit, w.completed_at
FROM tracking.workout_sets ws
JOIN tracking.workouts w ON w.id = ws.workout_id
WHERE w.user_id = $1 AND w.completed_at IS NOT NULL AND w.deleted_at IS NULL AND ws.load_value IS NOT NULL AND NOT ws.bodyweight
ORDER BY ws.exercise_id, w.completed_at DESC, ws.block_position DESC, ws.exercise_position DESC, ws.set_number DESC;

-- name: ListExerciseLoads :many
-- The sets with a load of an exercise in a user's newest finished workouts that logged one (at
-- most workout_limit workouts), newest first, the sets in session order. Weight added to
-- bodyweight isn't such a load.
WITH recent AS (
    SELECT w.id, w.session_name, w.completed_at
    FROM tracking.workouts w
    WHERE w.user_id = $1 AND w.completed_at IS NOT NULL AND w.deleted_at IS NULL
      AND EXISTS (
          SELECT 1 FROM tracking.workout_sets s
          WHERE s.workout_id = w.id AND s.exercise_id = $2 AND s.load_value IS NOT NULL AND NOT s.bodyweight
      )
    ORDER BY w.completed_at DESC
    LIMIT sqlc.arg(workout_limit)
)
SELECT r.id AS workout_id, r.session_name, r.completed_at, ws.set_number, ws.reps_value, ws.reps_unit, ws.per_side, ws.load_value, ws.load_unit
FROM recent r
JOIN tracking.workout_sets ws ON ws.workout_id = r.id
WHERE ws.exercise_id = $2 AND ws.load_value IS NOT NULL AND NOT ws.bodyweight
ORDER BY r.completed_at DESC, ws.block_position, ws.exercise_position, ws.set_number;
