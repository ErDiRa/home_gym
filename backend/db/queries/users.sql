-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, created_at FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT
    u.id,
    u.email,
    u.created_at,
    EXISTS (
        SELECT 1 FROM admins a WHERE a.user_id = u.id AND a.deleted_at IS NULL
    ) AS is_admin
FROM users u
WHERE u.id = $1 AND u.deleted_at IS NULL;

-- name: IsAdmin :one
SELECT EXISTS (
    SELECT 1
    FROM admins a
    JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL
    WHERE a.user_id = $1 AND a.deleted_at IS NULL
);
