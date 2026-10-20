-- name: ListTrainings :many
-- A user's trainings with their size, the active one first, then the newest.
SELECT
    t.id,
    t.name,
    t.description,
    t.active,
    t.start_date,
    t.end_date,
    t.weeks,
    t.updated_at,
    (SELECT count(*) FROM sessions s WHERE s.training_id = t.id)::bigint AS session_count,
    (SELECT count(*) FROM training_week_sessions ws WHERE ws.training_id = t.id)::bigint AS planned_count
FROM trainings t
WHERE t.user_id = $1 AND t.deleted_at IS NULL
ORDER BY t.active DESC, t.start_date DESC, t.name;

-- name: GetTraining :one
-- Only the user's own trainings: another user's is not found.
SELECT id, name, description, active, start_date, end_date, weeks, created_at, updated_at
FROM trainings
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListTrainingSessions :many
SELECT id, name, notes FROM sessions
WHERE training_id = sqlc.arg(training_id)::uuid
ORDER BY position;

-- name: ListTrainingBlocks :many
SELECT sb.id, sb.session_id, sb.kind, sb.name, sb.sets,
       sb.reps_value, sb.reps_max, sb.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, sb.per_side,
       sb.load_value, sb.load_max, sb.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       sb.score_type, sb.time_seconds, sb.interval_seconds, sb.notes
FROM session_blocks sb
JOIN sessions s ON s.id = sb.session_id
LEFT JOIN rep_units ru ON ru.id = sb.reps_unit
LEFT JOIN load_units lu ON lu.id = sb.load_unit
WHERE s.training_id = sqlc.arg(training_id)::uuid
ORDER BY sb.session_id, sb.position;

-- name: ListTrainingExercises :many
SELECT se.block_id, se.exercise_id, e.name, e.type, se.sets,
       se.reps_value, se.reps_max, se.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, se.per_side,
       se.load_value, se.load_max, se.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       se.tempo, se.notes
FROM session_exercises se
JOIN exercises e ON e.id = se.exercise_id
JOIN session_blocks sb ON sb.id = se.block_id
JOIN sessions s ON s.id = sb.session_id
LEFT JOIN rep_units ru ON ru.id = se.reps_unit
LEFT JOIN load_units lu ON lu.id = se.load_unit
WHERE s.training_id = sqlc.arg(training_id)::uuid
ORDER BY se.block_id, se.position;

-- name: ListWeekSessions :many
-- The week plan of a training: its slots by week, in order.
SELECT id, week, position, session_id FROM training_week_sessions
WHERE training_id = $1
ORDER BY week, position;

-- name: CreateTraining :one
INSERT INTO trainings (user_id, name, description, active, start_date, weeks)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, active, start_date, end_date, weeks, created_at, updated_at;

-- name: UpdateTraining :one
-- Another user's training updates nothing (no row).
UPDATE trainings
SET name = $3, description = $4, active = $5, start_date = $6, weeks = $7, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id, name, description, active, start_date, end_date, weeks, created_at, updated_at;

-- name: DeleteTraining :execrows
UPDATE trainings SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: InsertSession :one
-- A training's session (training_id set) or a library template (training_id NULL).
INSERT INTO sessions (user_id, training_id, position, name, notes)
VALUES (sqlc.arg(user_id), sqlc.narg(training_id), sqlc.arg(position), sqlc.arg(name), sqlc.arg(notes))
RETURNING id;

-- name: UpdateTrainingSession :execrows
-- Matching training_id too means a session id from another training updates nothing (0 rows).
UPDATE sessions
SET position = sqlc.arg(position), name = sqlc.arg(name), notes = sqlc.arg(notes), updated_at = now()
WHERE id = sqlc.arg(id) AND training_id = sqlc.arg(training_id)::uuid;

-- name: DeleteTrainingSessionsExcept :exec
-- Removes the sessions dropped from the training (their slots go with them).
DELETE FROM sessions
WHERE training_id = sqlc.arg(training_id)::uuid AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: DeleteTrainingBlocks :exec
-- Clears all blocks of a training before they are re-inserted; exercises go with them (cascade).
DELETE FROM session_blocks sb
USING sessions s
WHERE s.id = sb.session_id AND s.training_id = sqlc.arg(training_id)::uuid;

-- name: InsertWeekSession :exec
INSERT INTO training_week_sessions (training_id, week, position, session_id)
VALUES ($1, $2, $3, $4);

-- name: UpdateWeekSession :execrows
-- Matching training_id too means a slot id from another training updates nothing (0 rows).
UPDATE training_week_sessions
SET week = $3, position = $4, session_id = $5
WHERE id = $1 AND training_id = $2;

-- name: DeleteWeekSessionsExcept :exec
-- Removes the slots dropped from the week plan.
DELETE FROM training_week_sessions
WHERE training_id = $1 AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: ListLibrarySessions :many
-- The user's session templates with their size, by name.
SELECT
    s.id,
    s.name,
    s.notes,
    s.updated_at,
    (SELECT count(*) FROM session_blocks sb WHERE sb.session_id = s.id)::bigint AS block_count,
    (SELECT count(*) FROM session_exercises se
     JOIN session_blocks sb ON sb.id = se.block_id
     WHERE sb.session_id = s.id)::bigint AS exercise_count
FROM sessions s
WHERE s.user_id = $1 AND s.training_id IS NULL AND s.deleted_at IS NULL
ORDER BY lower(s.name), s.created_at;

-- name: GetLibrarySession :one
-- Only the user's own live templates: another user's is not found.
SELECT id, name, notes, created_at, updated_at FROM sessions
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;

-- name: ListSessionBlocks :many
SELECT sb.id, sb.session_id, sb.kind, sb.name, sb.sets,
       sb.reps_value, sb.reps_max, sb.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, sb.per_side,
       sb.load_value, sb.load_max, sb.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       sb.score_type, sb.time_seconds, sb.interval_seconds, sb.notes
FROM session_blocks sb
LEFT JOIN rep_units ru ON ru.id = sb.reps_unit
LEFT JOIN load_units lu ON lu.id = sb.load_unit
WHERE sb.session_id = $1
ORDER BY sb.position;

-- name: ListSessionExercises :many
SELECT se.block_id, se.exercise_id, e.name, e.type, se.sets,
       se.reps_value, se.reps_max, se.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, se.per_side,
       se.load_value, se.load_max, se.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       se.tempo, se.notes
FROM session_exercises se
JOIN exercises e ON e.id = se.exercise_id
JOIN session_blocks sb ON sb.id = se.block_id
LEFT JOIN rep_units ru ON ru.id = se.reps_unit
LEFT JOIN load_units lu ON lu.id = se.load_unit
WHERE sb.session_id = $1
ORDER BY se.block_id, se.position;

-- name: UpdateLibrarySession :execrows
-- Another user's, a deleted template or a training's session updates nothing (0 rows).
UPDATE sessions SET name = $3, notes = $4, updated_at = now()
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;

-- name: DeleteSessionBlocks :exec
-- Clears the blocks of one session before they are re-inserted.
DELETE FROM session_blocks WHERE session_id = $1;

-- name: DeleteLibrarySession :execrows
UPDATE sessions SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;

-- name: InsertSessionBlock :one
-- sqlc.narg marks the parameters that may be NULL (what the block's kind doesn't share).
INSERT INTO session_blocks (
    session_id, position, kind, name, sets,
    reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit,
    score_type, time_seconds, interval_seconds, notes
) VALUES (
    sqlc.arg(session_id), sqlc.arg(position), sqlc.arg(kind), sqlc.arg(name), sqlc.narg(sets),
    sqlc.narg(reps_value), sqlc.narg(reps_max), sqlc.narg(reps_unit), sqlc.arg(per_side),
    sqlc.narg(load_value), sqlc.narg(load_max), sqlc.narg(load_unit),
    sqlc.narg(score_type), sqlc.narg(time_seconds), sqlc.narg(interval_seconds), sqlc.arg(notes)
)
RETURNING id;

-- name: InsertSessionExercise :execrows
-- Only one of the user's own live exercises is inserted; any other inserts nothing (0 rows).
INSERT INTO session_exercises (
    block_id, exercise_id, position, sets,
    reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit, tempo, notes
)
SELECT
    sqlc.arg(block_id), e.id, sqlc.arg(position), sqlc.narg(sets),
    sqlc.narg(reps_value), sqlc.narg(reps_max), sqlc.narg(reps_unit), sqlc.arg(per_side),
    sqlc.narg(load_value), sqlc.narg(load_max), sqlc.narg(load_unit), sqlc.arg(tempo), sqlc.arg(notes)
FROM exercises e
WHERE e.id = sqlc.arg(exercise_id) AND e.user_id = sqlc.arg(user_id) AND e.deleted_at IS NULL;

-- name: ListBlockKinds :many
SELECT code, name, description, min_exercises, max_exercises, shared_sets, shared_reps, shared_load, scored
FROM block_kinds
ORDER BY position;

-- name: ListRepUnits :many
-- Oldest first, so the seeded defaults (reps, s, ...) lead and new units are appended.
SELECT id, name, min_value, max_value FROM rep_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: ListLoadUnits :many
-- Oldest first, like ListRepUnits.
SELECT id, name, min_value, max_value FROM load_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;
