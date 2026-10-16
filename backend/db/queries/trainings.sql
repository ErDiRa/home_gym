-- name: ListTrainings :many
SELECT
    t.id,
    t.name,
    t.description,
    t.active,
    t.updated_at,
    u.id AS client_id,
    u.first_name AS client_first_name,
    u.last_name AS client_last_name,
    (SELECT count(*) FROM training_units tu WHERE tu.training_id = t.id)::bigint AS unit_count,
    (SELECT count(*) FROM training_exercises te
     JOIN training_blocks tb ON tb.id = te.block_id
     JOIN training_units tu ON tu.id = tb.unit_id
     WHERE tu.training_id = t.id)::bigint AS exercise_count
FROM trainings t
JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL
WHERE t.deleted_at IS NULL
ORDER BY u.last_name, u.first_name, t.active DESC, t.updated_at DESC;

-- name: ListClientTrainings :many
SELECT
    t.id,
    t.name,
    t.active,
    t.updated_at,
    (SELECT count(*) FROM training_units tu WHERE tu.training_id = t.id)::bigint AS unit_count
FROM trainings t
WHERE t.user_id = $1 AND t.deleted_at IS NULL
ORDER BY t.active DESC, t.updated_at DESC;

-- name: GetTraining :one
-- Only trainings of live clients.
SELECT t.id, t.user_id AS client_id, t.name, t.description, t.active, t.created_at, t.updated_at
FROM trainings t
JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- name: ListTrainingUnits :many
SELECT id, name, notes, weekdays, dates FROM training_units
WHERE training_id = $1
ORDER BY position;

-- name: ListTrainingBlocks :many
SELECT tb.id, tb.unit_id, tb.name, tb.notes
FROM training_blocks tb
JOIN training_units tu ON tu.id = tb.unit_id
WHERE tu.training_id = $1
ORDER BY tb.unit_id, tb.position;

-- name: ListTrainingExercises :many
SELECT te.block_id, te.exercise_id, e.name, e.type, te.sets, te.reps, te.notes
FROM training_exercises te
JOIN exercises e ON e.id = te.exercise_id
JOIN training_blocks tb ON tb.id = te.block_id
JOIN training_units tu ON tu.id = tb.unit_id
WHERE tu.training_id = $1
ORDER BY te.block_id, te.position;

-- name: CreateTraining :one
INSERT INTO trainings (user_id, name, description, active)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id AS client_id, name, description, active, created_at, updated_at;

-- name: UpdateTraining :one
UPDATE trainings
SET user_id = $2, name = $3, description = $4, active = $5, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, user_id AS client_id, name, description, active, created_at, updated_at;

-- name: DeactivateOtherTrainings :exec
-- Run before activating a training, so the client keeps at most one active one.
UPDATE trainings SET active = false, updated_at = now()
WHERE user_id = $1 AND id <> $2 AND active AND deleted_at IS NULL;

-- name: DeleteTraining :execrows
UPDATE trainings SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: InsertTrainingUnit :one
INSERT INTO training_units (training_id, position, name, notes, weekdays, dates)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: UpdateTrainingUnit :execrows
UPDATE training_units
SET position = $3, name = $4, notes = $5, weekdays = $6, dates = $7
WHERE id = $1 AND training_id = $2;

-- name: DeleteTrainingUnitsExcept :exec
-- Removes the units dropped from the training.
DELETE FROM training_units
WHERE training_id = $1 AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: DeleteTrainingBlocks :exec
DELETE FROM training_blocks tb
USING training_units tu
WHERE tu.id = tb.unit_id AND tu.training_id = $1;

-- name: InsertTrainingBlock :one
INSERT INTO training_blocks (unit_id, position, name, notes)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: InsertBlockExercises :exec
-- The arrays are parallel: element i of each describes the exercise at position i.
INSERT INTO training_exercises (block_id, exercise_id, position, sets, reps, notes)
SELECT sqlc.arg(block_id), (sqlc.arg(exercise_ids)::uuid[])[i], i,
       (sqlc.arg(sets)::int[])[i], (sqlc.arg(reps)::text[])[i], (sqlc.arg(notes)::text[])[i]
FROM generate_subscripts(sqlc.arg(exercise_ids)::uuid[], 1) AS i;
