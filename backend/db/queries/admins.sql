-- name: GetAdminByEmail :one
SELECT id, email, password_hash, created_at FROM admins
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetAdminByID :one
SELECT id, email, created_at FROM admins
WHERE id = $1 AND deleted_at IS NULL;

-- name: IsAdmin :one
SELECT EXISTS (
    SELECT 1 FROM admins WHERE id = $1 AND deleted_at IS NULL
);
