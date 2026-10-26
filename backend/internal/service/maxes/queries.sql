-- name: ListExerciseMaxes :many
-- A user's 1RM history of their live exercises, newest first per exercise: the first entry of an
-- exercise is its current max.
SELECT m.id, m.exercise_id, m.load_value, m.load_unit, lu.name AS load_unit_name, m.achieved_on,
       m.workout_set_id, m.notes, m.created_at
FROM exercise_maxes m
JOIN exercises e ON e.id = m.exercise_id AND e.deleted_at IS NULL
JOIN load_units lu ON lu.id = m.load_unit
WHERE m.user_id = $1 AND m.deleted_at IS NULL
ORDER BY m.exercise_id, m.achieved_on DESC, m.created_at DESC;

-- name: GetLoadUnit :one
-- A live load unit; kg_per_unit is NULL unless it is a weight unit.
SELECT name, kg_per_unit FROM load_units WHERE id = $1 AND deleted_at IS NULL;

-- name: UserOwnsSet :one
-- Whether the set was logged for the exercise in one of the user's live workouts.
SELECT EXISTS (
    SELECT 1 FROM workout_sets ws
    JOIN workouts w ON w.id = ws.workout_id AND w.user_id = sqlc.arg(user_id) AND w.deleted_at IS NULL
    WHERE ws.id = sqlc.arg(id) AND ws.exercise_id = sqlc.arg(exercise_id)
);

-- name: CreateExerciseMax :one
-- Adds a 1RM to the history of one of the user's exercises. Another user's (or a deleted)
-- exercise inserts nothing, so no row comes back.
WITH m AS (
    INSERT INTO exercise_maxes (user_id, exercise_id, load_value, load_unit, achieved_on, workout_set_id, notes)
    SELECT e.user_id, e.id, sqlc.arg(load_value), sqlc.arg(load_unit), sqlc.arg(achieved_on),
           sqlc.narg(workout_set_id), sqlc.arg(notes)
    FROM exercises e
    WHERE e.id = sqlc.arg(exercise_id) AND e.user_id = sqlc.arg(user_id) AND e.deleted_at IS NULL
    RETURNING *
)
SELECT m.id, m.exercise_id, m.load_value, m.load_unit, lu.name AS load_unit_name, m.achieved_on,
       m.workout_set_id, m.notes, m.created_at
FROM m
JOIN load_units lu ON lu.id = m.load_unit;

-- name: DeleteExerciseMax :execrows
UPDATE exercise_maxes SET deleted_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
