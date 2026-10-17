-- name: ListUsers :many
SELECT
    u.id,
    u.email,
    u.first_name,
    u.last_name,
    u.created_at,
    (SELECT count(*) FROM trainings t WHERE t.user_id = u.id AND t.deleted_at IS NULL)::bigint AS training_count,
    -- At most one active training per client (unique index), so this returns one row at most.
    COALESCE((
        SELECT t.name FROM trainings t WHERE t.user_id = u.id AND t.active AND t.deleted_at IS NULL
    ), '')::text AS active_training
FROM users u
WHERE u.deleted_at IS NULL
ORDER BY u.last_name, u.first_name, u.email;

-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, last_name, created_at;

-- name: GetUserByID :one
SELECT id, email, first_name, last_name, created_at FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users
SET email = $2, first_name = $3, last_name = $4, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, email, first_name, last_name, created_at;

-- name: DeleteUser :execrows
-- Soft delete: the row stays, so trainings and ratings keep their references.
UPDATE users SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListClientTrainings :many
-- A client's trainings for their detail page, the active one first, then the latest.
SELECT
    t.id,
    t.name,
    t.active,
    t.start_date,
    t.end_date,
    t.updated_at,
    (SELECT count(*) FROM training_units tu WHERE tu.training_id = t.id)::bigint AS unit_count
FROM trainings t
WHERE t.user_id = $1 AND t.deleted_at IS NULL
ORDER BY t.active DESC, t.start_date DESC;

-- name: ListClientExerciseRatings :many
-- How one client rated exercises, for admins picking exercises for their training.
SELECT f.exercise_id, f.rating, f.notes, f.updated_at
FROM exercise_feedbacks f
JOIN users u ON u.id = f.user_id AND u.deleted_at IS NULL
JOIN exercises e ON e.id = f.exercise_id AND e.deleted_at IS NULL
WHERE f.user_id = $1 AND f.deleted_at IS NULL;

-- name: AdminEmailExists :one
-- Clients can't take an admin's email: login checks users before admins, so the admin would
-- be locked out.
SELECT EXISTS (
    SELECT 1 FROM admins WHERE email = $1 AND deleted_at IS NULL
);
