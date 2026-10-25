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

-- A user's cycle: a planned period of whole weeks from a Monday (like a training) with goals,
-- e.g. "Mobility, 8 weeks: better hips, better elbow extension". It groups trainings (each in at
-- most one cycle, see trainings.cycle_id); trainings without a cycle work on their own. Its
-- period is set on the cycle, not derived from its trainings, so a cycle can be planned before its
-- trainings arrive (e.g. a weekly CrossFit plan imported week by week).
CREATE TABLE cycles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    start_date  DATE NOT NULL CONSTRAINT cycles_start_monday CHECK (extract(isodow FROM start_date) = 1),
    weeks       INT NOT NULL CHECK (weeks BETWEEN 1 AND 52),
    end_date    DATE NOT NULL GENERATED ALWAYS AS (start_date + weeks * 7 - 1) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_cycles_user ON cycles (user_id) WHERE deleted_at IS NULL;

-- The goals of a cycle, in order: a short title ("Better hips"), optionally something measurable
-- to check at the end (target, e.g. "Deep squat hold 2 min"), and how it went. Rewritten when the
-- cycle is saved, like a training's blocks.
CREATE TABLE cycle_goals (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id   UUID NOT NULL REFERENCES cycles (id) ON DELETE CASCADE,
    position   INT NOT NULL,
    title      TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'achieved', 'missed')),
    notes      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_cycle_goals_cycle ON cycle_goals (cycle_id, position);

-- A user's training plan. A user can have several (e.g. one per phase, or strength, warm-ups and
-- mobility side by side); the active ones are those the tracker offers, each independently. It runs over whole weeks (e.g. a 4-week block) from a
-- Monday, so its weeks are calendar weeks; end_date is the last Sunday. Each week plans its own
-- sessions (training_week_sessions), done on any days of the week. It may belong to a cycle; it
-- may run partly or wholly outside the cycle's weeks (the cycle can be moved afterwards).
CREATE TABLE trainings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cycle_id    UUID REFERENCES cycles (id) ON DELETE SET NULL, -- NULL = not part of a cycle
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active      BOOLEAN NOT NULL DEFAULT false,
    start_date  DATE NOT NULL CONSTRAINT trainings_start_monday CHECK (extract(isodow FROM start_date) = 1),
    weeks       INT NOT NULL CHECK (weeks BETWEEN 1 AND 52),
    end_date    DATE NOT NULL GENERATED ALWAYS AS (start_date + weeks * 7 - 1) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_trainings_user ON trainings (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_trainings_cycle ON trainings (cycle_id) WHERE deleted_at IS NULL;

-- A session (e.g. "Session 1") is made of blocks (e.g. "Warm-up", "Superset"), and a block of
-- one or more exercises. Without a training_id it is a template in the user's library; with one
-- it is that training's own copy (adding a template to a training copies it, so later edits on
-- either side don't affect the other). A training's sessions keep their id when it is edited
-- (week plans and workouts point at them); blocks and their exercises are rewritten on every save.
CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    training_id UUID REFERENCES trainings (id) ON DELETE CASCADE, -- NULL = library template
    position    INT NOT NULL DEFAULT 0, -- order within the training
    name        TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ -- library templates only; a training's sessions are deleted
);

CREATE INDEX idx_sessions_training ON sessions (training_id, position);
CREATE INDEX idx_sessions_library ON sessions (user_id) WHERE training_id IS NULL AND deleted_at IS NULL;

-- The plan of a training's weeks: the sessions of week 1..weeks, in order (up to 7 a week; a
-- session may appear several times). A slot keeps its id when the training is edited, so the
-- workout started from it marks exactly this slot as done.
CREATE TABLE training_week_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_id UUID NOT NULL REFERENCES trainings (id) ON DELETE CASCADE,
    week        INT NOT NULL CHECK (week >= 1),
    position    INT NOT NULL CHECK (position BETWEEN 1 AND 7),
    session_id  UUID NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    UNIQUE (training_id, week, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_training_week_sessions_session ON training_week_sessions (session_id);

-- Kind of a training block. The rules that set a kind apart live in its row (how many
-- exercises it takes, whether reps and load are set once on the block for all exercises, whether
-- the block is done for one score instead of set by set), so a new kind is a new row. The rows
-- are part of the schema: the app needs them.
CREATE TABLE block_kinds (
    code          TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    position      INT NOT NULL,
    min_exercises INT NOT NULL CHECK (min_exercises >= 1),
    max_exercises INT CHECK (max_exercises >= min_exercises), -- NULL = no limit
    shared_sets   BOOLEAN NOT NULL,
    shared_reps   BOOLEAN NOT NULL,
    shared_load   BOOLEAN NOT NULL,
    -- A scored block (a WOD) is done as a whole for a score (see session_blocks.score_type)
    -- instead of set by set; its shared sets are rounds and optional.
    scored        BOOLEAN NOT NULL DEFAULT false
);

INSERT INTO block_kinds (code, name, description, position, min_exercises, max_exercises, shared_sets, shared_reps, shared_load, scored) VALUES
    ('single', 'Single', 'Exercises done one after another: all sets of one before the next, each with its own sets', 1, 1, NULL, false, false, false, false),
    ('superset', 'Superset', 'Exercises done alternating: same sets, own reps and load', 2, 2, NULL, true, false, false, false),
    ('complex', 'Complex', 'Exercises done back-to-back with the same weight: same sets, reps and load', 3, 2, NULL, true, true, true, false),
    ('wod', 'WOD', 'Workout of the day: the exercises done for one score (time, rounds, reps or calories)', 4, 1, NULL, true, false, false, true);

-- Units a planned amount is measured in: rep_units for reps (reps, seconds, metres, calories...),
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
    ('reps', 1, NULL), ('s', 1, NULL), ('min', NULL, NULL), ('m', NULL, NULL), ('cal', 1, NULL);

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
-- A scored block (a WOD) also has a score_type and its time (all NULL on other blocks):
--   for_time      time_seconds = optional time cap      score: the time (or reps, if capped)
--   amrap         time_seconds = duration (required)    score: rounds + reps
--   max_reps      time_seconds = optional duration      score: reps
--   max_calories  time_seconds = optional duration      score: calories
--   emom          time_seconds = duration (required), interval_seconds = every (e.g. 60)
--                                                       score: rounds (intervals) completed
CREATE TABLE session_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
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
    score_type       TEXT CHECK (score_type IN ('for_time', 'amrap', 'max_reps', 'max_calories', 'emom')),
    time_seconds     INT CHECK (time_seconds BETWEEN 1 AND 14400),
    interval_seconds INT CHECK (interval_seconds BETWEEN 1 AND 14400),
    notes      TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value)),
    CHECK (score_type IS NOT NULL OR (time_seconds IS NULL AND interval_seconds IS NULL)),
    CHECK (interval_seconds IS NULL OR score_type = 'emom')
);

CREATE INDEX idx_session_blocks_session ON session_blocks (session_id, position);

CREATE TABLE session_exercises (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id    UUID NOT NULL REFERENCES session_blocks (id) ON DELETE CASCADE,
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

CREATE INDEX idx_session_exercises_block ON session_exercises (block_id, position);

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

-- A performed session: started from a slot of a training's week plan, done on any day of that
-- week. completed_at is NULL while it is in progress (it can be resumed); a user has at most one
-- in progress. The training, its plan and the session may be edited or deleted later, so the
-- session's name is kept as it was.
CREATE TABLE workouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    training_id     UUID REFERENCES trainings (id) ON DELETE SET NULL,
    week_session_id UUID REFERENCES training_week_sessions (id) ON DELETE SET NULL,
    session_id      UUID REFERENCES sessions (id) ON DELETE SET NULL,
    session_name    TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    notes           TEXT NOT NULL DEFAULT '',
    deleted_at      TIMESTAMPTZ,
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX idx_workouts_user_completed ON workouts (user_id, completed_at) WHERE deleted_at IS NULL;
-- Several workouts can be in progress at once (e.g. a warm-up and the strength session), but a
-- slot of the week plan has at most one.
CREATE UNIQUE INDEX idx_workouts_slot_in_progress ON workouts (week_session_id)
    WHERE completed_at IS NULL AND deleted_at IS NULL;

-- One performed set of a workout. It points at the exercise rather than at session_exercises
-- (rewritten on every save) and keeps the exercise's place in the session (block and position
-- within the block, both from 1) for grouping. Reps and load are what was actually done, in
-- the same units as planned; either may be missing (a set logged partly), not both.
CREATE TABLE workout_sets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id        UUID NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    exercise_id       UUID NOT NULL REFERENCES exercises (id) ON DELETE RESTRICT,
    block_position    INT NOT NULL CHECK (block_position >= 1),
    exercise_position INT NOT NULL CHECK (exercise_position >= 1),
    set_number        INT NOT NULL CHECK (set_number BETWEEN 1 AND 100),
    reps_value        NUMERIC(7, 2),
    reps_unit         UUID NOT NULL REFERENCES rep_units (id) ON DELETE RESTRICT,
    per_side          BOOLEAN NOT NULL DEFAULT false,
    load_value        NUMERIC(7, 2),
    load_unit         UUID REFERENCES load_units (id) ON DELETE RESTRICT,
    completed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (reps_value IS NOT NULL OR load_value IS NOT NULL),
    UNIQUE (workout_id, block_position, exercise_position, set_number)
);

CREATE INDEX idx_workout_sets_exercise ON workout_sets (exercise_id);

-- The score of a scored block (a WOD) of a workout, at most one per block. Like workout_sets it
-- keeps the block's place in the session rather than pointing at it, and the format as planned
-- (score_type), so it stays readable when the session changes. Which of time/rounds/reps/calories
-- are set depends on the format (see session_blocks); rx is false when the WOD was scaled.
CREATE TABLE workout_results (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id     UUID NOT NULL REFERENCES workouts (id) ON DELETE CASCADE,
    block_position INT NOT NULL CHECK (block_position >= 1),
    score_type     TEXT NOT NULL CHECK (score_type IN ('for_time', 'amrap', 'max_reps', 'max_calories', 'emom')),
    time_seconds   INT CHECK (time_seconds BETWEEN 1 AND 86400),
    rounds         INT CHECK (rounds BETWEEN 0 AND 10000),
    reps           INT CHECK (reps BETWEEN 0 AND 100000),
    calories       INT CHECK (calories BETWEEN 0 AND 100000),
    rx             BOOLEAN NOT NULL DEFAULT true,
    notes          TEXT NOT NULL DEFAULT '',
    completed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (time_seconds IS NOT NULL OR rounds IS NOT NULL OR reps IS NOT NULL OR calories IS NOT NULL),
    UNIQUE (workout_id, block_position)
);
