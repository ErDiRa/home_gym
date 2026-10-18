-- name: ListTrainings :many
-- A user's trainings with their size, the active one first, then the newest.
SELECT
    t.id,
    t.name,
    t.description,
    t.active,
    t.start_date,
    t.end_date,
    t.sessions_per_week,
    t.updated_at,
    (SELECT count(*) FROM training_units tu WHERE tu.training_id = t.id)::bigint AS unit_count,
    (SELECT count(*) FROM training_exercises te
     JOIN training_blocks tb ON tb.id = te.block_id
     JOIN training_units tu ON tu.id = tb.unit_id
     WHERE tu.training_id = t.id)::bigint AS exercise_count
FROM trainings t
WHERE t.user_id = $1 AND t.deleted_at IS NULL
ORDER BY t.active DESC, t.start_date DESC, t.name;

-- name: GetTraining :one
-- Only the user's own trainings: another user's is not found.
SELECT id, name, description, active, start_date, end_date, sessions_per_week, created_at, updated_at
FROM trainings
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListTrainingUnits :many
SELECT id, name, notes FROM training_units
WHERE training_id = $1
ORDER BY position;

-- name: ListTrainingBlocks :many
SELECT tb.id, tb.unit_id, tb.kind, tb.name, tb.sets,
       tb.reps_value, tb.reps_max, tb.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, tb.per_side,
       tb.load_value, tb.load_max, tb.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       tb.notes
FROM training_blocks tb
JOIN training_units tu ON tu.id = tb.unit_id
LEFT JOIN rep_units ru ON ru.id = tb.reps_unit
LEFT JOIN load_units lu ON lu.id = tb.load_unit
WHERE tu.training_id = $1
ORDER BY tb.unit_id, tb.position;

-- name: ListTrainingExercises :many
SELECT te.block_id, te.exercise_id, e.name, e.type, te.sets,
       te.reps_value, te.reps_max, te.reps_unit, COALESCE(ru.name, '')::text AS reps_unit_name, te.per_side,
       te.load_value, te.load_max, te.load_unit, COALESCE(lu.name, '')::text AS load_unit_name,
       te.tempo, te.notes
FROM training_exercises te
JOIN exercises e ON e.id = te.exercise_id
JOIN training_blocks tb ON tb.id = te.block_id
JOIN training_units tu ON tu.id = tb.unit_id
LEFT JOIN rep_units ru ON ru.id = te.reps_unit
LEFT JOIN load_units lu ON lu.id = te.load_unit
WHERE tu.training_id = $1
ORDER BY te.block_id, te.position;

-- name: CreateTraining :one
INSERT INTO trainings (user_id, name, description, active, start_date, end_date, sessions_per_week)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, description, active, start_date, end_date, sessions_per_week, created_at, updated_at;

-- name: UpdateTraining :one
-- Another user's training updates nothing (no row).
UPDATE trainings
SET name = $3, description = $4, active = $5, start_date = $6, end_date = $7, sessions_per_week = $8, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id, name, description, active, start_date, end_date, sessions_per_week, created_at, updated_at;

-- name: DeactivateUserTrainings :exec
-- Run before saving a training as active, so the user keeps at most one active one.
UPDATE trainings SET active = false, updated_at = now()
WHERE user_id = $1 AND active AND deleted_at IS NULL;

-- name: DeleteTraining :execrows
UPDATE trainings SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: InsertTrainingUnit :one
INSERT INTO training_units (training_id, position, name, notes)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: UpdateTrainingUnit :execrows
-- Matching training_id too means a unit id from another training updates nothing (0 rows).
UPDATE training_units
SET position = $3, name = $4, notes = $5
WHERE id = $1 AND training_id = $2;

-- name: DeleteTrainingUnitsExcept :exec
-- Removes the units dropped from the training.
DELETE FROM training_units
WHERE training_id = $1 AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: DeleteTrainingBlocks :exec
-- Clears all blocks of a training before they are re-inserted; exercises go with them (cascade).
DELETE FROM training_blocks tb
USING training_units tu
WHERE tu.id = tb.unit_id AND tu.training_id = $1;

-- name: InsertTrainingBlock :one
-- sqlc.narg marks the parameters that may be NULL (what the block's kind doesn't share).
INSERT INTO training_blocks (
    unit_id, position, kind, name, sets,
    reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit, notes
) VALUES (
    sqlc.arg(unit_id), sqlc.arg(position), sqlc.arg(kind), sqlc.arg(name), sqlc.narg(sets),
    sqlc.narg(reps_value), sqlc.narg(reps_max), sqlc.narg(reps_unit), sqlc.arg(per_side),
    sqlc.narg(load_value), sqlc.narg(load_max), sqlc.narg(load_unit), sqlc.arg(notes)
)
RETURNING id;

-- name: InsertTrainingExercise :execrows
-- Only one of the user's own live exercises is inserted; any other inserts nothing (0 rows).
INSERT INTO training_exercises (
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
SELECT code, name, description, min_exercises, max_exercises, shared_sets, shared_reps, shared_load
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

-- name: GetActiveTrainingID :one
-- The training the tracker shows: the user's active one (at most one).
SELECT id FROM trainings
WHERE user_id = $1 AND active AND deleted_at IS NULL;
