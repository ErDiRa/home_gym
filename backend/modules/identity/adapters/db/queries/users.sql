-- name: GetUserByEmail :one
-- For login; emails are stored lower-cased.
SELECT id, email, password_hash FROM identity.users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email FROM identity.users
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
-- Users are created by hand; this is for tests (there is no sign-up). The email is stored
-- lower-cased, the password as a bcrypt hash.
INSERT INTO identity.users (email, first_name, last_name, password_hash)
VALUES (lower(sqlc.arg(email)), sqlc.arg(first_name), sqlc.arg(last_name), sqlc.arg(password_hash))
RETURNING id;
