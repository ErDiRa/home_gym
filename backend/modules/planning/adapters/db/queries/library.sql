-- The session library: templates are sessions without a training (LibraryRepository).

-- name: GetLibrarySession :one
-- Only the user's own live templates: another user's is not found.
SELECT id, name, notes, created_at, updated_at FROM planning.sessions
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;

-- name: ListSessionBlocks :many
SELECT sb.id, sb.session_id, sb.kind, sb.name, sb.sets,
       sb.reps_value, sb.reps_max, sb.reps_unit, sb.per_side, sb.load_value, sb.load_max, sb.load_unit,
       sb.score_type, sb.time_seconds, sb.interval_seconds, sb.notes
FROM planning.session_blocks sb
WHERE sb.session_id = $1
ORDER BY sb.position;

-- name: ListSessionExercises :many
SELECT se.block_id, se.exercise_id, se.sets,
       se.reps_value, se.reps_max, se.reps_unit, se.per_side, se.load_value, se.load_max, se.load_unit,
       se.tempo, se.notes
FROM planning.session_exercises se
JOIN planning.session_blocks sb ON sb.id = se.block_id
WHERE sb.session_id = $1
ORDER BY se.block_id, se.position;

-- name: UpdateLibrarySession :execrows
-- Another user's, a deleted template or a training's session updates nothing (0 rows).
UPDATE planning.sessions SET name = $3, notes = $4, updated_at = now()
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;

-- name: DeleteSessionBlocks :exec
-- Clears the blocks of one session before they are re-inserted.
DELETE FROM planning.session_blocks WHERE session_id = $1;

-- name: DeleteLibrarySession :execrows
UPDATE planning.sessions SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND training_id IS NULL AND deleted_at IS NULL;
