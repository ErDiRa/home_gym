-- The identity module: the users who sign in. Users are created by hand (no sign-up); every other
-- module's rows belong to one of them (user_id).
--
-- Every table has a nullable deleted_at for soft deletion: NULL means the row is live.
-- Queries must filter on `deleted_at IS NULL`. Uniqueness is enforced with partial
-- indexes so that soft-deleted rows do not block re-creating the same data.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS identity;

-- crypt() and gen_salt() for creating users by hand (bcrypt password_hash), e.g. the dev seed.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE identity.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_email ON identity.users (email) WHERE deleted_at IS NULL;
