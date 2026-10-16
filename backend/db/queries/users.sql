-- name: CreateUser :one
INSERT INTO users (email, first_name, last_name, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, last_name, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, first_name, last_name, created_at FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: UserExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL
);

-- name: ListUsers :many
SELECT
    u.id,
    u.email,
    u.first_name,
    u.last_name,
    u.created_at,
    (SELECT count(*) FROM trainings t WHERE t.user_id = u.id AND t.deleted_at IS NULL)::bigint AS training_count,
    COALESCE((
        SELECT t.name FROM trainings t WHERE t.user_id = u.id AND t.active AND t.deleted_at IS NULL
    ), '')::text AS active_training
FROM users u
WHERE u.deleted_at IS NULL
ORDER BY u.last_name, u.first_name, u.email;

-- name: UpdateUser :one
UPDATE users
SET email = $2, first_name = $3, last_name = $4, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, email, first_name, last_name, created_at;

-- name: DeleteUser :execrows
UPDATE users SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
