-- The contract's queries (api/module): what other modules read of the plans.

-- name: ListActiveTrainings :many
-- The trainings the tracker offers: the user's active ones with their cycle, by name.
SELECT t.id, t.name, t.start_date, t.end_date, t.weeks, t.cycle_id, c.name AS cycle_name
FROM planning.trainings t
LEFT JOIN planning.cycles c ON c.id = t.cycle_id AND c.deleted_at IS NULL
WHERE t.user_id = $1 AND t.active AND t.deleted_at IS NULL
ORDER BY lower(t.name), t.start_date;

-- name: ListWeekPlan :many
-- The sessions planned for one week of one of the user's trainings, in order.
SELECT ws.id, ws.session_id, s.name
FROM planning.training_week_sessions ws
JOIN planning.sessions s ON s.id = ws.session_id
JOIN planning.trainings t ON t.id = ws.training_id AND t.user_id = sqlc.arg(user_id)
WHERE ws.training_id = sqlc.arg(training_id) AND ws.week = sqlc.arg(week)
ORDER BY ws.position;

-- name: GetWeekSlot :one
-- A slot of the week plan of one of the user's live trainings, for starting a workout of it.
SELECT ws.id, ws.training_id, ws.session_id, s.name AS session_name
FROM planning.training_week_sessions ws
JOIN planning.sessions s ON s.id = ws.session_id
JOIN planning.trainings t ON t.id = ws.training_id AND t.user_id = $2 AND t.deleted_at IS NULL
WHERE ws.id = $1;

-- name: GetCycleOverview :one
-- One of the user's live cycles, for the tracker.
SELECT id, name, start_date, end_date, weeks
FROM planning.cycles
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListCycleGoalStatuses :many
SELECT title, target, status FROM planning.cycle_goals
WHERE cycle_id = $1
ORDER BY position;
