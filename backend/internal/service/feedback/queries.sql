-- name: UpsertExerciseFeedback :one
-- A client's rating of an exercise; rating again replaces it. The ON CONFLICT target repeats
-- the partial unique index's WHERE so Postgres can match that index.
INSERT INTO exercise_feedbacks (user_id, exercise_id, rating, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, exercise_id) WHERE deleted_at IS NULL
DO UPDATE SET rating = EXCLUDED.rating,
              notes = EXCLUDED.notes,
              updated_at = now()
RETURNING *;

-- name: GetExerciseStats :many
-- The exercises a user has logged, with how often and their own rating (NULL if unrated).
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
    COUNT(wl.id)::bigint AS times_performed,
    f.rating AS rating
FROM exercises e
JOIN workout_logs wl ON wl.exercise_id = e.id AND wl.user_id = $1 AND wl.deleted_at IS NULL
LEFT JOIN exercise_feedbacks f ON f.exercise_id = e.id AND f.user_id = $1 AND f.deleted_at IS NULL
WHERE e.deleted_at IS NULL
GROUP BY e.id, f.rating
ORDER BY times_performed DESC, e.name;

-- name: LogWorkout :one
INSERT INTO workout_logs (user_id, exercise_id, sets_done, reps_done, weight_kg)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
