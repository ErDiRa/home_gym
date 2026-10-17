-- name: GetUserByEmail :one
-- For login; emails are stored lower-cased.
SELECT id, email, password_hash, created_at FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, first_name, last_name, created_at FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAdminByEmail :one
SELECT id, email, password_hash, created_at FROM admins
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetAdminByID :one
SELECT id, email, created_at FROM admins
WHERE id = $1 AND deleted_at IS NULL;

-- name: IsAdmin :one
-- Checked on every /api/admin request by auth.RequireAdmin.
SELECT EXISTS (
    SELECT 1 FROM admins WHERE id = $1 AND deleted_at IS NULL
);

-- name: UserExists :one
-- Checked on every /api/client request by auth.RequireUser.
SELECT EXISTS (
    SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL
);
