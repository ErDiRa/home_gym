-- name: CreateExercise :one
INSERT INTO exercises (name, joint, muscle_group, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListExercises :many
SELECT * FROM exercises
ORDER BY name;

-- name: UpsertExerciseFeedback :one
INSERT INTO exercise_feedbacks (user_id, exercise_id, rating, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, exercise_id)
DO UPDATE SET rating = EXCLUDED.rating,
              notes = EXCLUDED.notes,
              updated_at = now()
RETURNING *;
