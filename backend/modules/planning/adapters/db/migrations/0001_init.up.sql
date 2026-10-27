-- The planning module: a user's cycles, trainings, their sessions and week plans, and the
-- session library (templates). Blocks and exercises reference the catalog's block kinds, units and
-- exercises.
--
-- Every table has a nullable deleted_at for soft deletion where rows are soft-deleted: NULL means
-- the row is live. Queries must filter on `deleted_at IS NULL`.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS planning;

-- A user's cycle: a planned period of whole weeks from a Monday (like a training) with goals,
-- e.g. "Mobility, 8 weeks: better hips, better elbow extension". It groups trainings (each in at
-- most one cycle, see trainings.cycle_id); trainings without a cycle work on their own. Its
-- period is set on the cycle, not derived from its trainings, so a cycle can be planned before its
-- trainings arrive (e.g. a weekly CrossFit plan imported week by week).
CREATE TABLE planning.cycles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    start_date  DATE NOT NULL CONSTRAINT cycles_start_monday CHECK (extract(isodow FROM start_date) = 1),
    weeks       INT NOT NULL CHECK (weeks BETWEEN 1 AND 52),
    end_date    DATE NOT NULL GENERATED ALWAYS AS (start_date + weeks * 7 - 1) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_cycles_user ON planning.cycles (user_id) WHERE deleted_at IS NULL;

-- The goals of a cycle, in order: a short title ("Better hips"), optionally something measurable
-- to check at the end (target, e.g. "Deep squat hold 2 min"), and how it went. Rewritten when the
-- cycle is saved, like a training's blocks.
CREATE TABLE planning.cycle_goals (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id   UUID NOT NULL REFERENCES planning.cycles (id) ON DELETE CASCADE,
    position   INT NOT NULL,
    title      TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'achieved', 'missed')),
    notes      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_cycle_goals_cycle ON planning.cycle_goals (cycle_id, position);

-- A user's training plan. A user can have several (e.g. one per phase, or strength, warm-ups and
-- mobility side by side); the active ones are those the tracker offers, each independently. It
-- runs over whole weeks (e.g. a 4-week block) from a Monday, so its weeks are calendar weeks;
-- end_date is the last Sunday. Each week plans its own sessions (training_week_sessions), done on
-- any days of the week. It may belong to a cycle; it may run partly or wholly outside the cycle's
-- weeks (the cycle can be moved afterwards).
CREATE TABLE planning.trainings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    cycle_id    UUID REFERENCES planning.cycles (id) ON DELETE SET NULL, -- NULL = not part of a cycle
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

CREATE INDEX idx_trainings_user ON planning.trainings (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_trainings_cycle ON planning.trainings (cycle_id) WHERE deleted_at IS NULL;

-- A session (e.g. "Session 1") is made of blocks (e.g. "Warm-up", "Superset"), and a block of
-- one or more exercises. Without a training_id it is a template in the user's library; with one
-- it is that training's own copy (adding a template to a training copies it, so later edits on
-- either side don't affect the other). A training's sessions keep their id when it is edited
-- (week plans and workouts point at them); blocks and their exercises are rewritten on every save.
CREATE TABLE planning.sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    training_id UUID REFERENCES planning.trainings (id) ON DELETE CASCADE, -- NULL = library template
    position    INT NOT NULL DEFAULT 0, -- order within the training
    name        TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ -- library templates only; a training's sessions are deleted
);

CREATE INDEX idx_sessions_training ON planning.sessions (training_id, position);
CREATE INDEX idx_sessions_library ON planning.sessions (user_id) WHERE training_id IS NULL AND deleted_at IS NULL;

-- The plan of a training's weeks: the sessions of week 1..weeks, in order (up to 7 a week; a
-- session may appear several times). A slot keeps its id when the training is edited, so the
-- workout started from it marks exactly this slot as done.
CREATE TABLE planning.training_week_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_id UUID NOT NULL REFERENCES planning.trainings (id) ON DELETE CASCADE,
    week        INT NOT NULL CHECK (week >= 1),
    position    INT NOT NULL CHECK (position BETWEEN 1 AND 7),
    session_id  UUID NOT NULL REFERENCES planning.sessions (id) ON DELETE CASCADE,
    UNIQUE (training_id, week, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_training_week_sessions_session ON planning.training_week_sessions (session_id);

-- Blocks and exercises both carry a planned amount: reps (a value or a range in a rep unit,
-- optionally per side) and load (a value or a range in a load unit), and so do sets. Whether
-- sets, reps and load sit on the block (for all its exercises) or on each exercise depends on
-- the block's kind (shared_sets / shared_reps / shared_load); the other side stays empty. A load
-- may name only its unit, without a value: the unit the sets are logged in.
-- A scored block (a WOD) also has a score_type and its time (all NULL on other blocks):
--   for_time      time_seconds = optional time cap      score: the time (or reps, if capped)
--   amrap         time_seconds = duration (required)    score: rounds + reps
--   max_reps      time_seconds = optional duration      score: reps
--   max_calories  time_seconds = optional duration      score: calories
--   emom          time_seconds = duration (required), interval_seconds = every (e.g. 60)
--                                                       score: rounds (intervals) completed
CREATE TABLE planning.session_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES planning.sessions (id) ON DELETE CASCADE,
    position   INT NOT NULL,
    kind       TEXT NOT NULL REFERENCES catalog.block_kinds (code) ON DELETE RESTRICT,
    name       TEXT NOT NULL,
    sets       INT CHECK (sets BETWEEN 1 AND 100),
    reps_value NUMERIC(7, 2),
    reps_max   NUMERIC(7, 2),
    reps_unit  UUID REFERENCES catalog.rep_units (id) ON DELETE RESTRICT,
    per_side   BOOLEAN NOT NULL DEFAULT false,
    load_value NUMERIC(7, 2),
    load_max   NUMERIC(7, 2),
    load_unit  UUID REFERENCES catalog.load_units (id) ON DELETE RESTRICT,
    score_type       TEXT CHECK (score_type IN ('for_time', 'amrap', 'max_reps', 'max_calories', 'emom')),
    time_seconds     INT CHECK (time_seconds BETWEEN 1 AND 14400),
    interval_seconds INT CHECK (interval_seconds BETWEEN 1 AND 14400),
    notes      TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK (load_value IS NULL OR load_unit IS NOT NULL),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value)),
    CHECK (score_type IS NOT NULL OR (time_seconds IS NULL AND interval_seconds IS NULL)),
    CHECK (interval_seconds IS NULL OR score_type = 'emom')
);

CREATE INDEX idx_session_blocks_session ON planning.session_blocks (session_id, position);

CREATE TABLE planning.session_exercises (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id    UUID NOT NULL REFERENCES planning.session_blocks (id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE RESTRICT,
    position    INT NOT NULL,
    sets        INT CHECK (sets BETWEEN 1 AND 100),
    reps_value  NUMERIC(7, 2),
    reps_max    NUMERIC(7, 2),
    reps_unit   UUID REFERENCES catalog.rep_units (id) ON DELETE RESTRICT,
    per_side    BOOLEAN NOT NULL DEFAULT false,
    load_value  NUMERIC(7, 2),
    load_max    NUMERIC(7, 2),
    load_unit   UUID REFERENCES catalog.load_units (id) ON DELETE RESTRICT,
    tempo       TEXT NOT NULL DEFAULT '', -- e.g. '31X1': eccentric, pause, concentric, pause (X = explosive)
    notes       TEXT NOT NULL DEFAULT '',
    CHECK ((reps_value IS NULL) = (reps_unit IS NULL)),
    CHECK (reps_max IS NULL OR (reps_value IS NOT NULL AND reps_max > reps_value)),
    CHECK (load_value IS NULL OR load_unit IS NOT NULL),
    CHECK (load_max IS NULL OR (load_value IS NOT NULL AND load_max > load_value))
);

CREATE INDEX idx_session_exercises_block ON planning.session_exercises (block_id, position);
