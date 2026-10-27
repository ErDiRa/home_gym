-- Writing and loading cycles with their goals (CycleRepository).

-- name: GetCycle :one
-- Only the user's own live cycles: another user's is not found.
SELECT id, name, description, start_date, end_date, weeks, created_at, updated_at
FROM planning.cycles
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListCycleGoals :many
SELECT id, position, title, target, status, notes FROM planning.cycle_goals
WHERE cycle_id = $1
ORDER BY position;

-- name: ListCycleTrainings :many
-- The cycle's live trainings in the order they run.
SELECT id, name, active, start_date, end_date, weeks FROM planning.trainings
WHERE cycle_id = sqlc.arg(cycle_id)::uuid AND deleted_at IS NULL
ORDER BY start_date, lower(name);

-- name: CreateCycle :one
INSERT INTO planning.cycles (user_id, name, description, start_date, weeks)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, start_date, end_date, weeks, created_at, updated_at;

-- name: UpdateCycle :one
-- Another user's or a deleted cycle updates nothing (no row). Its trainings don't move.
UPDATE planning.cycles
SET name = $3, description = $4, start_date = $5, weeks = $6, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
RETURNING id, name, description, start_date, end_date, weeks, created_at, updated_at;

-- name: DeleteCycle :execrows
UPDATE planning.cycles SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: DeleteCycleGoals :exec
-- Clears a cycle's goals before they are re-inserted.
DELETE FROM planning.cycle_goals WHERE cycle_id = $1;

-- name: InsertCycleGoal :exec
INSERT INTO planning.cycle_goals (cycle_id, position, title, target, status, notes)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClearCycleFromTrainings :exec
-- Takes the trainings out of a cycle that is deleted (a soft delete doesn't fire ON DELETE SET NULL).
UPDATE planning.trainings SET cycle_id = NULL, updated_at = now()
WHERE cycle_id = sqlc.arg(cycle_id)::uuid;

-- name: DeleteCycleTrainings :exec
-- Soft-deletes the live trainings of a cycle that is deleted with them.
UPDATE planning.trainings SET deleted_at = now(), updated_at = now()
WHERE cycle_id = sqlc.arg(cycle_id)::uuid AND deleted_at IS NULL;
