-- The catalog module: what trainings are planned with. Each user's exercises with their links,
-- videos and ratings, and the lists shared by all users: joints, muscle groups, focuses, block
-- kinds and the units of reps and load.
--
-- Every table has a nullable deleted_at for soft deletion: NULL means the row is live.
-- Queries must filter on `deleted_at IS NULL`. Uniqueness is enforced with partial
-- indexes so that soft-deleted rows do not block re-creating the same data.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS catalog;

-- Reference lists an exercise picks from.
CREATE TABLE catalog.joints (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_joints_name ON catalog.joints (name) WHERE deleted_at IS NULL;

CREATE TABLE catalog.muscle_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_muscle_groups_name ON catalog.muscle_groups (name) WHERE deleted_at IS NULL;

-- Training focus an exercise belongs to, e.g. weightlifting, functional fitness, shoulder health.
-- Users can add new ones; like joints and muscle groups they are shared by all users.
CREATE TABLE catalog.focuses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    alias      TEXT NOT NULL, -- short label shown as a tag in tables; the name shows on hover
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_focuses_name ON catalog.focuses (lower(name)) WHERE deleted_at IS NULL;

-- A user's exercise library, which their trainings pick from.
CREATE TABLE catalog.exercises (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    type         TEXT NOT NULL CHECK (type IN ('mobility', 'strength')), -- mainly improves mobility or strength
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_exercises_user ON catalog.exercises (user_id) WHERE deleted_at IS NULL;

-- An exercise involves several joints and trains several muscle groups. These pure link
-- tables are hard-deleted (no deleted_at); the soft-deleted rows are the exercises, joints
-- and muscle groups themselves.
CREATE TABLE catalog.exercise_joints (
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    joint_id    UUID NOT NULL REFERENCES catalog.joints (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, joint_id)
);

CREATE TABLE catalog.exercise_muscle_groups (
    exercise_id     UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    muscle_group_id UUID NOT NULL REFERENCES catalog.muscle_groups (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, muscle_group_id)
);

-- Links to videos showing how to perform an exercise, in display order.
CREATE TABLE catalog.exercise_videos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    position    INT NOT NULL,
    url         TEXT NOT NULL
);

CREATE INDEX idx_exercise_videos_exercise ON catalog.exercise_videos (exercise_id, position);

CREATE TABLE catalog.exercise_focuses (
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    focus_id    UUID NOT NULL REFERENCES catalog.focuses (id) ON DELETE RESTRICT,
    PRIMARY KEY (exercise_id, focus_id)
);

-- Kind of a training block. The rules that set a kind apart live in its row (how many
-- exercises it takes, whether reps and load are set once on the block for all exercises, whether
-- the block is done for one score instead of set by set), so a new kind is a new row. The rows
-- are part of the schema: the app needs them.
CREATE TABLE catalog.block_kinds (
    code          TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    position      INT NOT NULL,
    min_exercises INT NOT NULL CHECK (min_exercises >= 1),
    max_exercises INT CHECK (max_exercises >= min_exercises), -- NULL = no limit
    shared_sets   BOOLEAN NOT NULL,
    shared_reps   BOOLEAN NOT NULL,
    shared_load   BOOLEAN NOT NULL,
    -- A scored block (a WOD) is done as a whole for a score (see planning.session_blocks.score_type)
    -- instead of set by set; its shared sets are rounds and optional.
    scored        BOOLEAN NOT NULL DEFAULT false
);

INSERT INTO catalog.block_kinds (code, name, description, position, min_exercises, max_exercises, shared_sets, shared_reps, shared_load, scored) VALUES
    ('single', 'Single', 'Exercises done one after another: all sets of one before the next, each with its own sets', 1, 1, NULL, false, false, false, false),
    ('superset', 'Superset', 'Exercises done alternating: same sets, own reps and load', 2, 2, NULL, true, false, false, false),
    ('complex', 'Complex', 'Exercises done back-to-back with the same weight: same sets, reps and load', 3, 2, NULL, true, true, true, false),
    ('wod', 'WOD', 'Workout of the day: the exercises done for one score (time, rounds, reps or calories)', 4, 1, NULL, true, false, false, true);

-- Units a planned amount is measured in: rep_units for reps (reps, seconds, metres, calories...),
-- load_units for load (kg, % 1RM, RPE...). Users can add new ones. min/max_value bound the
-- values planned in the unit (NULL = unbounded).
CREATE TABLE catalog.rep_units (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    min_value  NUMERIC(7, 2),
    max_value  NUMERIC(7, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), -- keeps insertion order within one INSERT
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_rep_units_name ON catalog.rep_units (lower(name)) WHERE deleted_at IS NULL;

INSERT INTO catalog.rep_units (name, min_value, max_value) VALUES
    ('reps', 1, NULL), ('s', 1, NULL), ('min', NULL, NULL), ('m', NULL, NULL), ('cal', 1, NULL);

-- Weight units carry their size in kg (kg_per_unit), so a load converts between them (kg <-> lb);
-- % 1RM (percent_of_max) is a share of the exercise's 1RM, converted with the user's max. Units
-- users add have neither. cal is what is logged per set on a machine interval planned by time
-- (e.g. 4 min on the Echo Bike, logged as 52 cal).
CREATE TABLE catalog.load_units (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    min_value      NUMERIC(7, 2),
    max_value      NUMERIC(7, 2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(), -- keeps insertion order within one INSERT
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    kg_per_unit    NUMERIC(10, 8) CHECK (kg_per_unit > 0),
    percent_of_max BOOLEAN NOT NULL DEFAULT false,
    CHECK (NOT (percent_of_max AND kg_per_unit IS NOT NULL))
);

CREATE UNIQUE INDEX idx_load_units_name ON catalog.load_units (lower(name)) WHERE deleted_at IS NULL;

INSERT INTO catalog.load_units (name, min_value, max_value, kg_per_unit, percent_of_max) VALUES
    ('kg', 0, NULL, 1, false), ('lb', 0, NULL, 0.45359237, false), ('% 1RM', 0, 200, NULL, true),
    ('RPE', 1, 10, NULL, false), ('RIR', 0, 10, NULL, false), ('cal', 0, NULL, NULL, false);

-- A user's rating (1-5) and notes for an exercise, at most one live row per user and
-- exercise. The exercise picker shows and filters by them.
CREATE TABLE catalog.exercise_feedbacks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    rating      INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    notes       TEXT NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_exercise_feedbacks_user_exercise
    ON catalog.exercise_feedbacks (user_id, exercise_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_exercise_feedbacks_exercise ON catalog.exercise_feedbacks (exercise_id);
