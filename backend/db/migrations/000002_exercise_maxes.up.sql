-- Weight units carry their size in kg, so a load converts between them (kg <-> lb); % 1RM is a
-- share of the exercise's 1RM, converted with the user's max. Units users add have neither.
ALTER TABLE load_units
    ADD COLUMN kg_per_unit    NUMERIC(10, 8) CHECK (kg_per_unit > 0),
    ADD COLUMN percent_of_max BOOLEAN NOT NULL DEFAULT false,
    ADD CHECK (NOT (percent_of_max AND kg_per_unit IS NOT NULL));

UPDATE load_units SET kg_per_unit = 1 WHERE lower(name) = 'kg' AND deleted_at IS NULL;
UPDATE load_units SET kg_per_unit = 0.45359237 WHERE lower(name) IN ('lb', 'lbs') AND deleted_at IS NULL;
UPDATE load_units SET percent_of_max = true WHERE name = '% 1RM' AND deleted_at IS NULL;

-- A user's 1RM history per exercise, entered by hand or saved from a logged single
-- (workout_set_id). The newest entry (achieved_on, then created_at) is the current max that
-- % 1RM loads are calculated from; the load is always in a weight unit.
CREATE TABLE exercise_maxes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    exercise_id    UUID NOT NULL REFERENCES exercises (id) ON DELETE CASCADE,
    load_value     NUMERIC(7, 2) NOT NULL CHECK (load_value > 0),
    load_unit      UUID NOT NULL REFERENCES load_units (id) ON DELETE RESTRICT,
    achieved_on    DATE NOT NULL,
    workout_set_id UUID REFERENCES workout_sets (id) ON DELETE SET NULL,
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_exercise_maxes_user ON exercise_maxes (user_id, exercise_id, achieved_on DESC)
    WHERE deleted_at IS NULL;
