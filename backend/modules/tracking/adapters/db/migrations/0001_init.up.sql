-- The tracking module: what a user did. Workouts started from a slot of a planning week plan, their
-- sets and WOD scores, and the 1RM history of the catalog's exercises.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS tracking;

-- A performed session: started from a slot of a training's week plan, done on any day of that
-- week. completed_at is NULL while it is in progress (it can be resumed). The training, its plan
-- and the session may be edited or deleted later, so the session's name is kept as it was.
CREATE TABLE tracking.workouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    training_id     UUID REFERENCES planning.trainings (id) ON DELETE SET NULL,
    week_session_id UUID REFERENCES planning.training_week_sessions (id) ON DELETE SET NULL,
    session_id      UUID REFERENCES planning.sessions (id) ON DELETE SET NULL,
    session_name    TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    notes           TEXT NOT NULL DEFAULT '',
    deleted_at      TIMESTAMPTZ,
    CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX idx_workouts_user_completed ON tracking.workouts (user_id, completed_at) WHERE deleted_at IS NULL;
-- Several workouts can be in progress at once (e.g. a warm-up and the strength session), but a
-- slot of the week plan has at most one.
CREATE UNIQUE INDEX idx_workouts_slot_in_progress ON tracking.workouts (week_session_id)
    WHERE completed_at IS NULL AND deleted_at IS NULL;

-- One performed set of a workout. It points at the exercise rather than at the planned session
-- exercise (rewritten on every save) and keeps the exercise's place in the session (block and
-- position within the block, both from 1) for grouping. Reps and load are what was actually done,
-- in the same units as planned; either may be missing (a set logged partly), not both.
CREATE TABLE tracking.workout_sets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id        UUID NOT NULL REFERENCES tracking.workouts (id) ON DELETE CASCADE,
    exercise_id       UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE RESTRICT,
    block_position    INT NOT NULL CHECK (block_position >= 1),
    exercise_position INT NOT NULL CHECK (exercise_position >= 1),
    set_number        INT NOT NULL CHECK (set_number BETWEEN 1 AND 100),
    reps_value        NUMERIC(7, 2),
    reps_unit         UUID NOT NULL REFERENCES catalog.rep_units (id) ON DELETE RESTRICT,
    per_side          BOOLEAN NOT NULL DEFAULT false,
    load_value        NUMERIC(7, 2),
    load_unit         UUID REFERENCES catalog.load_units (id) ON DELETE RESTRICT,
    completed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((load_value IS NULL) = (load_unit IS NULL)),
    CHECK (reps_value IS NOT NULL OR load_value IS NOT NULL),
    UNIQUE (workout_id, block_position, exercise_position, set_number)
);

CREATE INDEX idx_workout_sets_exercise ON tracking.workout_sets (exercise_id);

-- The score of a scored block (a WOD) of a workout, at most one per block. Like workout_sets it
-- keeps the block's place in the session rather than pointing at it, and the format as planned
-- (score_type), so it stays readable when the session changes. Which of time/rounds/reps/calories
-- are set depends on the format (see planning.session_blocks); rx is false when the WOD was scaled.
CREATE TABLE tracking.workout_results (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id     UUID NOT NULL REFERENCES tracking.workouts (id) ON DELETE CASCADE,
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

-- A user's 1RM history per exercise, entered by hand or saved from a logged single
-- (workout_set_id). The newest entry (achieved_on, then created_at) is the current max that
-- % 1RM loads are calculated from; the load is always in a weight unit.
CREATE TABLE tracking.exercise_maxes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    exercise_id    UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE CASCADE,
    load_value     NUMERIC(7, 2) NOT NULL CHECK (load_value > 0),
    load_unit      UUID NOT NULL REFERENCES catalog.load_units (id) ON DELETE RESTRICT,
    achieved_on    DATE NOT NULL,
    workout_set_id UUID REFERENCES tracking.workout_sets (id) ON DELETE SET NULL,
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_exercise_maxes_user ON tracking.exercise_maxes (user_id, exercise_id, achieved_on DESC)
    WHERE deleted_at IS NULL;
