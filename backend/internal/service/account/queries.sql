-- name: GetUserByEmail :one
-- For login; emails are stored lower-cased.
SELECT id, email, password_hash, created_at FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, first_name, last_name, created_at FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: UserExists :one
-- Checked on every /api request by auth.RequireUser.
SELECT EXISTS (
    SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL
);
