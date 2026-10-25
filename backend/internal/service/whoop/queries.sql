-- name: GetConnection :one
SELECT * FROM whoop_connections WHERE user_id = $1;

-- name: SaveConnection :exec
-- Connecting again (or refreshing the tokens) replaces the stored tokens.
INSERT INTO whoop_connections (user_id, access_token, refresh_token, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET access_token = EXCLUDED.access_token, refresh_token = EXCLUDED.refresh_token, expires_at = EXCLUDED.expires_at;

-- name: DeleteConnection :exec
DELETE FROM whoop_connections WHERE user_id = $1;

-- name: UserOwnsWorkout :one
SELECT EXISTS (SELECT 1 FROM workouts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL);

-- name: GetWorkoutData :one
SELECT d.* FROM workout_whoop_data d
JOIN workouts w ON w.id = d.workout_id AND w.user_id = $2 AND w.deleted_at IS NULL
WHERE d.workout_id = $1;

-- name: SaveWorkoutData :one
-- Linking again (another activity, or the same one once scored) replaces the stored numbers.
INSERT INTO workout_whoop_data (
    workout_id, whoop_workout_id, sport_name, start_at, end_at, score_state,
    strain, avg_hr, max_hr, kilojoule, zone_ms, recovery_score, resting_hr, hrv_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (workout_id) DO UPDATE
SET whoop_workout_id = EXCLUDED.whoop_workout_id, sport_name = EXCLUDED.sport_name,
    start_at = EXCLUDED.start_at, end_at = EXCLUDED.end_at, score_state = EXCLUDED.score_state,
    strain = EXCLUDED.strain, avg_hr = EXCLUDED.avg_hr, max_hr = EXCLUDED.max_hr,
    kilojoule = EXCLUDED.kilojoule, zone_ms = EXCLUDED.zone_ms, recovery_score = EXCLUDED.recovery_score,
    resting_hr = EXCLUDED.resting_hr, hrv_ms = EXCLUDED.hrv_ms, fetched_at = now()
RETURNING *;

-- name: DeleteWorkoutData :execrows
DELETE FROM workout_whoop_data d
USING workouts w
WHERE d.workout_id = $1 AND w.id = d.workout_id AND w.user_id = $2;
