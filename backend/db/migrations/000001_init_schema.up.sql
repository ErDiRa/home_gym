CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Every table has a nullable deleted_at for soft deletion: NULL means the row is live.
-- Queries must filter on `deleted_at IS NULL`. Uniqueness is enforced with partial
-- indexes so that soft-deleted rows do not block re-creating the same data.

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    first_name    TEXT NOT NULL,
    last_name     TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;

-- Admins are separate accounts with their own credentials; they are not linked to users.
CREATE TABLE admins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_admins_email ON admins (email) WHERE deleted_at IS NULL;

-- Reference lists an exercise picks from.
CREATE TABLE joints (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_joints_name ON joints (name) WHERE deleted_at IS NULL;

CREATE TABLE muscle_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_muscle_groups_name ON muscle_groups (name) WHERE deleted_at IS NULL;

-- Training focus an exercise belongs to, e.g. weightlifting, functional fitness, shoulder health.
-- Admins can add new ones.
CREATE TABLE focuses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    alias      TEXT NOT NULL, -- short label shown as a tag in tables; the name shows on hover
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_focuses_name ON focuses (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE exercises (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    video_url    TEXT NOT NULL DEFAULT '', -- link to a video showing how to perform the exercise
    type         TEXT NOT NULL CHECK (type IN ('mobility', 'strength')), -- mainly improves mobility or strength
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

-- An exercise involves several joints and trains several muscle groups. These pure link
-- tables are hard-deleted (no deleted_at); the soft-deleted rows are the exercises, joints
-- and muscle groups themselves.
CREATE TABLE exercise_joints (
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    joint_id    UUID NOT NULL REFERENCES joints (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, joint_id)
);

CREATE TABLE exercise_muscle_groups (
    exercise_id     UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    muscle_group_id UUID NOT NULL REFERENCES muscle_groups (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, muscle_group_id)
);

CREATE TABLE exercise_focuses (
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    focus_id    UUID NOT NULL REFERENCES focuses (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, focus_id)
);

-- A training is written for one client. A client can have several (e.g. one per phase), of
-- which at most one is active.
CREATE TABLE trainings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_trainings_user ON trainings (user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_trainings_user_active ON trainings (user_id) WHERE active AND deleted_at IS NULL;

-- A training is made of units (e.g. "Day A"), a unit of blocks (e.g. "Warm-up", "Superset"),
-- and a block of one or more exercises. A unit is done on recurring weekdays (ISO, 1 = Monday)
-- and/or specific dates. Units keep their id when a training is edited; blocks and their
-- exercises are rewritten on every save.
CREATE TABLE training_units (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_id UUID NOT NULL REFERENCES trainings (id) ON DELETE CASCADE,
    position    INT NOT NULL,
    name        TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    weekdays    SMALLINT[] NOT NULL DEFAULT '{}' CHECK (weekdays <@ ARRAY[1, 2, 3, 4, 5, 6, 7]::smallint[]),
    dates       DATE[] NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_training_units_training ON training_units (training_id, position);

-- Kind of a training block. The rules that set a kind apart live in its row (how many
-- exercises it takes, whether reps and load are set once on the block for all exercises), so
-- a new kind is a new row. The rows are part of the schema: the app needs them.
CREATE TABLE block_kinds (
    code          TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    position      INT NOT NULL,
    min_exercises INT NOT NULL CHECK (min_exercises >= 1),
    max_exercises INT CHECK (max_exercises >= min_exercises), -- NULL = no limit
    shared_reps   BOOLEAN NOT NULL,
    shared_load   BOOLEAN NOT NULL
);

INSERT INTO block_kinds (code, name, description, position, min_exercises, max_exercises, shared_reps, shared_load) VALUES
    ('single', 'Single', 'One exercise on its own', 1, 1, 1, false, false),
    ('superset', 'Superset', 'Exercises done alternating: same sets, own reps and load', 2, 2, NULL, false, false),
    ('complex', 'Complex', 'Exercises done back-to-back with the same weight: same sets, reps and load', 3, 2, NULL, true, true);

-- Units a planned amount is measured in: rep_units for reps (reps, seconds, metres...),
-- load_units for load (kg, % 1RM, RPE...). Admins can add new ones. min/max_value bound the
-- values planned in the unit (NULL = unbounded).
CREATE TABLE rep_units (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    min_value  NUMERIC(7, 2),
    max_value  NUMERIC(7, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), -- keeps insertion order within one INSERT
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_rep_units_name ON rep_units (lower(name)) WHERE deleted_at IS NULL;

INSERT INTO rep_units (name, min_value, max_value) VALUES
    ('reps', 1, NULL), ('s', 1, NULL), ('min', NULL, NULL), ('m', NULL, NULL);

CREATE TABLE load_units (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    min_value  NUMERIC(7, 2),
    max_value  NUMERIC(7, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), -- keeps insertion order within one INSERT
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_load_units_name ON load_units (lower(name)) WHERE deleted_at IS NULL;

INSERT INTO load_units (name, min_value, max_value) VALUES
    ('kg', 0, NULL), ('lb', 0, NULL), ('% 1RM', 0, 200), ('RPE', 1, 10), ('RIR', 0, 10);

-- Blocks and exercises both carry a planned amount: reps (a value or a range in a rep unit,
-- optionally per side) and load (a value or a range in a load unit). A block's sets apply to
-- all of its exercises. Whether reps and load sit on the block or on each exercise depends on
-- the block's kind (shared_reps / shared_load); the other side stays empty.
CREATE TABLE training_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id    UUID NOT NULL REFERENCES training_units (id) ON DELETE CASCADE,
    position   INT NOT NULL,
    kind       TEXT NOT NULL REFERENCES block_kinds (code) ON DELETE RESTRICT,
    name       TEXT NOT NULL,
    sets       INT NOT NULL CHECK (sets BETWEEN 1 AND 100),
    reps_value NUMERIC(7, 2),
    reps_max   NUMERIC(7, 2),
    reps_unit  UUID REFERENCES rep_units (id) ON DELETE RESTRICT,
    per_side   BOOLEAN NOT NULL DEFAULT false,
    load_value NUMERIC(7, 2),
    load_max   NUMERIC(7, 2),
    load_unit  UUID REFERENCES load_units (id) ON DELETE RESTRICT,
    notes      TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value))
);

CREATE INDEX idx_training_blocks_unit ON training_blocks (unit_id, position);

CREATE TABLE training_exercises (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id    UUID NOT NULL REFERENCES training_blocks (id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    position    INT NOT NULL,
    reps_value  NUMERIC(7, 2),
    reps_max    NUMERIC(7, 2),
    reps_unit   UUID REFERENCES rep_units (id) ON DELETE RESTRICT,
    per_side    BOOLEAN NOT NULL DEFAULT false,
    load_value  NUMERIC(7, 2),
    load_max    NUMERIC(7, 2),
    load_unit   UUID REFERENCES load_units (id) ON DELETE RESTRICT,
    notes       TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value))
);

CREATE INDEX idx_training_exercises_block ON training_exercises (block_id, position);

CREATE TABLE exercise_feedbacks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    rating      INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    notes       TEXT NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_exercise_feedbacks_user_exercise
    ON exercise_feedbacks (user_id, exercise_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_exercise_feedbacks_exercise ON exercise_feedbacks (exercise_id);

CREATE TABLE workout_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    exercise_id  UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sets_done    INT NOT NULL CHECK (sets_done > 0),
    reps_done    INT NOT NULL CHECK (reps_done > 0),
    weight_kg    NUMERIC(6, 2) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_workout_logs_user_exercise ON workout_logs (user_id, exercise_id);
