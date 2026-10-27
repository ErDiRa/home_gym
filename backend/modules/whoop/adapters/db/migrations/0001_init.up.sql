-- The whoop module: a user's connection to their WHOOP account and the WHOOP activity linked to
-- their tracking workouts.

-- Created by the migration runner already; declared here for sqlc, which reads this file.
CREATE SCHEMA IF NOT EXISTS whoop;

-- A user's connection to their WHOOP account (OAuth tokens). The refresh token rotates: every
-- refresh returns a new one, which replaces the stored one.
CREATE TABLE whoop.whoop_connections (
    user_id       UUID PRIMARY KEY REFERENCES identity.users (id) ON DELETE CASCADE,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    connected_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The WHOOP activity the user linked to a workout, with its numbers and the recovery of that day
-- as fetched when linking. The numbers are null while WHOOP hasn't scored the activity yet
-- (score_state 'PENDING_SCORE'); linking again fetches them anew.
CREATE TABLE whoop.workout_whoop_data (
    workout_id       UUID PRIMARY KEY REFERENCES tracking.workouts (id) ON DELETE CASCADE,
    whoop_workout_id UUID NOT NULL,
    sport_name       TEXT NOT NULL,
    start_at         TIMESTAMPTZ NOT NULL,
    end_at           TIMESTAMPTZ NOT NULL,
    score_state      TEXT NOT NULL,
    strain           NUMERIC(7, 2),
    avg_hr           INT,
    max_hr           INT,
    kilojoule        NUMERIC(9, 2),
    zone_ms          INT[], -- time in heart-rate zones 0 to 5, in milliseconds
    recovery_score   INT,
    resting_hr       INT,
    hrv_ms           NUMERIC(7, 2),
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
