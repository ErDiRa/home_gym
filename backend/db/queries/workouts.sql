-- name: LogWorkout :one
INSERT INTO workout_logs (user_id, exercise_id, sets_done, reps_done, weight_kg)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetExerciseStats :many
SELECT
    e.id AS exercise_id,
    e.name,
    e.joint,
    e.muscle_group,
    COUNT(wl.id)::bigint AS times_performed,
    f.rating AS rating
FROM exercises e
JOIN workout_logs wl ON wl.exercise_id = e.id AND wl.user_id = $1 AND wl.deleted_at IS NULL
LEFT JOIN exercise_feedbacks f ON f.exercise_id = e.id AND f.user_id = $1 AND f.deleted_at IS NULL
WHERE e.deleted_at IS NULL
GROUP BY e.id, f.rating
ORDER BY times_performed DESC, e.name;
