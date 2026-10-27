-- The read models: the lists, mapped straight to the HTTP responses.

-- name: ListTrainings :many
-- A user's trainings with their size and cycle, the active one first, then the newest.
SELECT
    t.id,
    t.cycle_id,
    c.name AS cycle_name,
    t.name,
    t.description,
    t.active,
    t.start_date,
    t.end_date,
    t.weeks,
    t.updated_at,
    (SELECT count(*) FROM planning.sessions s WHERE s.training_id = t.id)::bigint AS session_count,
    (SELECT count(*) FROM planning.training_week_sessions ws WHERE ws.training_id = t.id)::bigint AS planned_count
FROM planning.trainings t
LEFT JOIN planning.cycles c ON c.id = t.cycle_id AND c.deleted_at IS NULL
WHERE t.user_id = $1 AND t.deleted_at IS NULL
ORDER BY t.active DESC, t.start_date DESC, t.name;

-- name: ListLibrarySessions :many
-- The user's session templates with their size, by name.
SELECT
    s.id,
    s.name,
    s.notes,
    s.updated_at,
    (SELECT count(*) FROM planning.session_blocks sb WHERE sb.session_id = s.id)::bigint AS block_count,
    (SELECT count(*) FROM planning.session_exercises se
     JOIN planning.session_blocks sb ON sb.id = se.block_id
     WHERE sb.session_id = s.id)::bigint AS exercise_count
FROM planning.sessions s
WHERE s.user_id = $1 AND s.training_id IS NULL AND s.deleted_at IS NULL
ORDER BY lower(s.name), s.created_at;

-- name: ListCycles :many
-- A user's live cycles with their size, the latest first.
SELECT
    c.id,
    c.name,
    c.description,
    c.start_date,
    c.end_date,
    c.weeks,
    c.updated_at,
    (SELECT count(*) FROM planning.cycle_goals g WHERE g.cycle_id = c.id)::bigint AS goal_count,
    (SELECT count(*) FROM planning.trainings t WHERE t.cycle_id = c.id AND t.deleted_at IS NULL)::bigint AS training_count
FROM planning.cycles c
WHERE c.user_id = $1 AND c.deleted_at IS NULL
ORDER BY c.start_date DESC, lower(c.name);
