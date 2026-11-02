-- A note on an exercise as it was done in a workout (e.g. "BikeErg instead, avg HR 132"): one per
-- exercise at its place in the session, which it keeps rather than pointing at it, like the sets.
CREATE TABLE tracking.workout_exercise_notes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id        UUID NOT NULL REFERENCES tracking.workouts (id) ON DELETE CASCADE,
    exercise_id       UUID NOT NULL REFERENCES catalog.exercises (id),
    block_position    INT NOT NULL CHECK (block_position >= 1),
    exercise_position INT NOT NULL CHECK (exercise_position >= 1),
    notes             TEXT NOT NULL CHECK (notes <> ''),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workout_id, block_position, exercise_position)
);
