-- name: ListBlockKinds :many
SELECT code, name, description, min_exercises, max_exercises, shared_sets, shared_reps, shared_load
FROM block_kinds
ORDER BY position;

-- name: ListRepUnits :many
SELECT id, name, min_value, max_value FROM rep_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: CreateRepUnit :one
INSERT INTO rep_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value;

-- name: ListLoadUnits :many
SELECT id, name, min_value, max_value FROM load_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: CreateLoadUnit :one
INSERT INTO load_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value;
