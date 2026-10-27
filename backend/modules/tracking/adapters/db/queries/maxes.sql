-- The 1RM history (MaxRepository).

-- name: CreateExerciseMax :one
-- The exercise is checked to be one of the user's live ones before (the catalog's contract).
INSERT INTO tracking.exercise_maxes (user_id, exercise_id, load_value, load_unit, achieved_on, workout_set_id, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UserOwnsSet :one
-- Whether the set was logged for the exercise in one of the user's live workouts.
SELECT EXISTS (
    SELECT 1 FROM tracking.workout_sets ws
    JOIN tracking.workouts w ON w.id = ws.workout_id AND w.user_id = sqlc.arg(user_id) AND w.deleted_at IS NULL
    WHERE ws.id = sqlc.arg(id) AND ws.exercise_id = sqlc.arg(exercise_id)
);

-- name: DeleteExerciseMax :execrows
UPDATE tracking.exercise_maxes SET deleted_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
