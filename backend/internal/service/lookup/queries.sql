-- name: ListJoints :many
SELECT id, name FROM joints
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListMuscleGroups :many
SELECT id, name FROM muscle_groups
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListFocuses :many
SELECT id, name, alias FROM focuses
WHERE deleted_at IS NULL
ORDER BY name;

-- name: CreateFocus :one
INSERT INTO focuses (name, alias) VALUES ($1, $2)
RETURNING id, name, alias;

-- name: ListBlockKinds :many
SELECT code, name, description, min_exercises, max_exercises, shared_sets, shared_reps, shared_load, scored
FROM block_kinds
ORDER BY position;

-- name: ListRepUnits :many
-- Oldest first, so the seeded defaults (reps, s, ...) lead and new units are appended.
SELECT id, name, min_value, max_value FROM rep_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: CreateRepUnit :one
INSERT INTO rep_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value;

-- name: ListLoadUnits :many
-- Oldest first, like ListRepUnits.
SELECT id, name, min_value, max_value FROM load_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: CreateLoadUnit :one
INSERT INTO load_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value;
