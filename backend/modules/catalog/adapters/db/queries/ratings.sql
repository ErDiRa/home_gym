-- name: UpsertExerciseFeedback :one
-- A user's rating of one of their exercises; rating again replaces it. Another user's (or a
-- deleted) exercise inserts nothing, so no row comes back. The ON CONFLICT target repeats the
-- partial unique index's WHERE so Postgres can match that index.
INSERT INTO catalog.exercise_feedbacks (user_id, exercise_id, rating, notes)
SELECT e.user_id, e.id, sqlc.arg(rating), sqlc.arg(notes)
FROM catalog.exercises e
WHERE e.id = sqlc.arg(exercise_id) AND e.user_id = sqlc.arg(user_id) AND e.deleted_at IS NULL
ON CONFLICT (user_id, exercise_id) WHERE deleted_at IS NULL
DO UPDATE SET rating = EXCLUDED.rating,
              notes = EXCLUDED.notes,
              updated_at = now()
RETURNING id, user_id, exercise_id, rating, notes, updated_at;
