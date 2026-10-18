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
-- Users can add new ones; like joints and muscle groups they are shared by all users.
CREATE TABLE focuses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    alias      TEXT NOT NULL, -- short label shown as a tag in tables; the name shows on hover
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_focuses_name ON focuses (lower(name)) WHERE deleted_at IS NULL;

-- A user's exercise library, which their trainings pick from.
CREATE TABLE exercises (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    type         TEXT NOT NULL CHECK (type IN ('mobility', 'strength')), -- mainly improves mobility or strength
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_exercises_user ON exercises (user_id) WHERE deleted_at IS NULL;

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

-- Links to videos showing how to perform an exercise, in display order.
CREATE TABLE exercise_videos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    position    INT NOT NULL,
    url         TEXT NOT NULL
);

CREATE INDEX idx_exercise_videos_exercise ON exercise_videos (exercise_id, position);

CREATE TABLE exercise_focuses (
    exercise_id UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    focus_id    UUID NOT NULL REFERENCES focuses (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, focus_id)
);

-- A user's training plan. A user can have several (e.g. one per phase), of which at most one
-- is active: the one the tracker shows. It runs over a period (e.g. a 4-week block), both days
-- included; the period is independent of the active flag. Instead of fixed days, it asks for a
-- number of sessions per week, done whenever there is time.
CREATE TABLE trainings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    active            BOOLEAN NOT NULL DEFAULT false,
    start_date        DATE NOT NULL,
    end_date          DATE NOT NULL,
    sessions_per_week INT NOT NULL DEFAULT 3 CHECK (sessions_per_week BETWEEN 1 AND 14),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT trainings_period_check CHECK (end_date >= start_date)
);

CREATE INDEX idx_trainings_user ON trainings (user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_trainings_user_active ON trainings (user_id) WHERE active AND deleted_at IS NULL;

-- A training is made of units (e.g. "Day A"), a unit of blocks (e.g. "Warm-up", "Superset"),
-- and a block of one or more exercises. Units are done in turn, one per session. Units keep
-- their id when a training is edited (workouts point at them); blocks and their exercises are
-- rewritten on every save.
CREATE TABLE training_units (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_id UUID NOT NULL REFERENCES trainings (id) ON DELETE CASCADE,
    position    INT NOT NULL,
    name        TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT ''
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
    shared_sets   BOOLEAN NOT NULL,
    shared_reps   BOOLEAN NOT NULL,
    shared_load   BOOLEAN NOT NULL
);

INSERT INTO block_kinds (code, name, description, position, min_exercises, max_exercises, shared_sets, shared_reps, shared_load) VALUES
    ('single', 'Single', 'Exercises done one after another: all sets of one before the next, each with its own sets', 1, 1, NULL, false, false, false),
    ('superset', 'Superset', 'Exercises done alternating: same sets, own reps and load', 2, 2, NULL, true, false, false),
    ('complex', 'Complex', 'Exercises done back-to-back with the same weight: same sets, reps and load', 3, 2, NULL, true, true, true);

-- Units a planned amount is measured in: rep_units for reps (reps, seconds, metres...),
-- load_units for load (kg, % 1RM, RPE...). Users can add new ones. min/max_value bound the
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
-- optionally per side) and load (a value or a range in a load unit), and so do sets. Whether
-- sets, reps and load sit on the block (for all its exercises) or on each exercise depends on
-- the block's kind (shared_sets / shared_reps / shared_load); the other side stays empty.
CREATE TABLE training_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id    UUID NOT NULL REFERENCES training_units (id) ON DELETE CASCADE,
    position   INT NOT NULL,
    kind       TEXT NOT NULL REFERENCES block_kinds (code) ON DELETE RESTRICT,
    name       TEXT NOT NULL,
    sets       INT CHECK (sets BETWEEN 1 AND 100),
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
    sets        INT CHECK (sets BETWEEN 1 AND 100),
    reps_value  NUMERIC(7, 2),
    reps_max    NUMERIC(7, 2),
    reps_unit   UUID REFERENCES rep_units (id) ON DELETE RESTRICT,
    per_side    BOOLEAN NOT NULL DEFAULT false,
    load_value  NUMERIC(7, 2),
    load_max    NUMERIC(7, 2),
    load_unit   UUID REFERENCES load_units (id) ON DELETE RESTRICT,
    tempo       TEXT NOT NULL DEFAULT '', -- e.g. '31X1': eccentric, pause, concentric, pause (X = explosive)
    notes       TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value))
);

CREATE INDEX idx_training_exercises_block ON training_exercises (block_id, position);

-- A user's rating (1-5) and notes for an exercise, at most one live row per user and
-- exercise. The exercise picker shows and filters by them.
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

-- A performed session: one unit of a training, done on any day. completed_at is NULL while
-- it is in progress (it can be resumed); a user has at most one in progress. The training and
-- unit may be edited or deleted later, so the unit's name is kept as it was.
CREATE TABLE workouts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    training_id  UUID REFERENCES trainings (id) ON DELETE SET NULL,
    unit_id      UUID REFERENCES training_units (id) ON DELETE SET NULL,
    unit_name    TEXT NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    notes        TEXT NOT NULL DEFAULT '',
    deleted_at   TIMESTAMPTZ,
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX idx_workouts_user_completed ON workouts (user_id, completed_at) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_workouts_user_in_progress ON workouts (user_id)
    WHERE completed_at IS NULL AND deleted_at IS NULL;

-- One performed set of a workout. It points at the exercise rather than at training_exercises
-- (rewritten on every save) and keeps the exercise's place in the unit (block and position
-- within the block, both from 1) for grouping. Reps and load are what was actually done, in
-- the same units as planned.
CREATE TABLE workout_sets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id        UUID NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    exercise_id       UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    block_position    INT NOT NULL CHECK (block_position >= 1),
    exercise_position INT NOT NULL CHECK (exercise_position >= 1),
    set_number        INT NOT NULL CHECK (set_number BETWEEN 1 AND 100),
    reps_value        NUMERIC(7, 2) NOT NULL,
    reps_unit         UUID NOT NULL REFERENCES rep_units (id) ON DELETE RESTRICT,
    per_side          BOOLEAN NOT NULL DEFAULT false,
    load_value        NUMERIC(7, 2),
    load_unit         UUID REFERENCES load_units (id) ON DELETE RESTRICT,
    completed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    UNIQUE (workout_id, block_position, exercise_position, set_number)
);

CREATE INDEX idx_workout_sets_exercise ON workout_sets (exercise_id);
