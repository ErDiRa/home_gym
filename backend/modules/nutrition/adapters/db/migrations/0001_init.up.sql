-- The nutrition module: a user's body (profile and weight history) and the macros they ate per day,
-- from which the daily targets are estimated.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS nutrition;

-- What the energy estimate needs of a user's body besides the weight (one row per user). The
-- activity factor covers the day outside training (1.2 at a desk, up to 2 on one's feet); the
-- workouts are added on top.
CREATE TABLE nutrition.profiles (
    user_id          UUID PRIMARY KEY REFERENCES identity.users (id) ON DELETE CASCADE,
    height_cm        NUMERIC(4, 1) NOT NULL CHECK (height_cm BETWEEN 100 AND 250),
    birth_date       DATE NOT NULL,
    sex              TEXT NOT NULL CHECK (sex IN ('male', 'female')),
    activity_factor  NUMERIC(3, 2) NOT NULL DEFAULT 1.3 CHECK (activity_factor BETWEEN 1.0 AND 2.0),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A user's weight history with the body fat when measured: at most one live entry per day; the
-- newest on or before a day is the weight of that day, the newest with body fat its body fat.
CREATE TABLE nutrition.body_weights (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    measured_on      DATE NOT NULL,
    weight_kg        NUMERIC(5, 2) NOT NULL CHECK (weight_kg BETWEEN 20 AND 400),
    body_fat_percent NUMERIC(3, 1) CHECK (body_fat_percent BETWEEN 3 AND 60),
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_body_weights_user_day ON nutrition.body_weights (user_id, measured_on) WHERE deleted_at IS NULL;

-- The macros a user ate on a day, as totals (at most one live row per day). Each may be left out,
-- not all; the energy follows from them (4/9/4 kcal per gram).
CREATE TABLE nutrition.daily_macros (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    day        DATE NOT NULL,
    protein_g  NUMERIC(5, 1) CHECK (protein_g BETWEEN 0 AND 2000),
    fat_g      NUMERIC(5, 1) CHECK (fat_g BETWEEN 0 AND 2000),
    carbs_g    NUMERIC(5, 1) CHECK (carbs_g BETWEEN 0 AND 2000),
    kcal       NUMERIC(6, 1) NOT NULL GENERATED ALWAYS AS
                   (COALESCE(protein_g, 0) * 4 + COALESCE(fat_g, 0) * 9 + COALESCE(carbs_g, 0) * 4) STORED,
    notes      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CHECK (protein_g IS NOT NULL OR fat_g IS NOT NULL OR carbs_g IS NOT NULL)
);

CREATE UNIQUE INDEX idx_daily_macros_user_day ON nutrition.daily_macros (user_id, day) WHERE deleted_at IS NULL;
