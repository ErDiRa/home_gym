-- Saved blocks: a user's reusable combinations of exercises (LibraryRepository, read model).

-- name: ListBlockTemplates :many
-- The user's live saved blocks by name.
SELECT id, name, kind, created_at, updated_at FROM planning.block_templates
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY lower(name);

-- name: GetBlockTemplate :one
-- Only the user's own live saved blocks: another user's is not found.
SELECT id, name, kind, created_at, updated_at FROM planning.block_templates
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListBlockTemplateExercises :many
-- The exercises of the saved blocks ids, in order.
SELECT template_id, exercise_id FROM planning.block_template_exercises
WHERE template_id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY template_id, position;

-- name: InsertBlockTemplate :one
INSERT INTO planning.block_templates (user_id, name, kind) VALUES ($1, $2, $3)
RETURNING id;

-- name: UpdateBlockTemplate :execrows
-- Another user's or a deleted saved block updates nothing (0 rows).
UPDATE planning.block_templates SET name = $3, kind = $4, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: DeleteBlockTemplateExercises :exec
-- Clears the exercises of a saved block before they are re-inserted.
DELETE FROM planning.block_template_exercises WHERE template_id = $1;

-- name: InsertBlockTemplateExercise :exec
INSERT INTO planning.block_template_exercises (template_id, position, exercise_id) VALUES ($1, $2, $3);

-- name: DeleteBlockTemplate :execrows
UPDATE planning.block_templates SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
