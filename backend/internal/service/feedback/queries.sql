-- name: UpsertExerciseFeedback :one
-- A user's rating of one of their exercises; rating again replaces it. Another user's (or a
-- deleted) exercise inserts nothing, so no row comes back. The ON CONFLICT target repeats the
-- partial unique index's WHERE so Postgres can match that index.
INSERT INTO exercise_feedbacks (user_id, exercise_id, rating, notes)
SELECT e.user_id, e.id, sqlc.arg(rating), sqlc.arg(notes)
FROM exercises e
WHERE e.id = sqlc.arg(exercise_id) AND e.user_id = sqlc.arg(user_id) AND e.deleted_at IS NULL
ON CONFLICT (user_id, exercise_id) WHERE deleted_at IS NULL
DO UPDATE SET rating = EXCLUDED.rating,
              notes = EXCLUDED.notes,
              updated_at = now()
RETURNING *;

-- name: ListExerciseRatings :many
-- How a user rated their exercises, for the exercise picker.
SELECT f.exercise_id, f.rating, f.notes, f.updated_at
FROM exercise_feedbacks f
JOIN exercises e ON e.id = f.exercise_id AND e.deleted_at IS NULL
WHERE f.user_id = $1 AND f.deleted_at IS NULL;

-- name: GetExerciseStats :many
-- The exercises a user has done in finished workouts, with in how many workouts and their
-- rating (NULL if unrated).
SELECT
    e.id AS exercise_id,
    e.name,
    ARRAY(
        SELECT j.name FROM exercise_joints ej
        JOIN joints j ON j.id = ej.joint_id AND j.deleted_at IS NULL
        WHERE ej.exercise_id = e.id ORDER BY j.name
    )::text[] AS joints,
    ARRAY(
        SELECT m.name FROM exercise_muscle_groups em
        JOIN muscle_groups m ON m.id = em.muscle_group_id AND m.deleted_at IS NULL
        WHERE em.exercise_id = e.id ORDER BY m.name
    )::text[] AS muscle_groups,
    COUNT(DISTINCT w.id)::bigint AS times_performed,
    f.rating AS rating
FROM exercises e
JOIN workout_sets ws ON ws.exercise_id = e.id
JOIN workouts w ON w.id = ws.workout_id AND w.user_id = $1 AND w.completed_at IS NOT NULL AND w.deleted_at IS NULL
LEFT JOIN exercise_feedbacks f ON f.exercise_id = e.id AND f.user_id = $1 AND f.deleted_at IS NULL
WHERE e.user_id = $1 AND e.deleted_at IS NULL
GROUP BY e.id, f.rating
ORDER BY times_performed DESC, e.name;
