-- Dev seed: sample clients with trainings (needs the schema's block kinds and units, and
-- seed-exercises.sql). Clients log in with password "test".
-- Skipped when anna@example.com already exists. Squat also gets a second video link.
--   anna@example.com  gym training: an active and an older inactive training, all block kinds,
--                     ranges, kg / % 1RM / RPE / RIR loads
--   ben@example.com   home training: kettlebell work, seconds, per-side reps, a unit on a date
--   clara@example.com no trainings, for empty states

-- Helpers, dropped with the session. Units and exercises are looked up by name.
CREATE FUNCTION pg_temp.exercise_id(ex_name text) RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE id uuid;
BEGIN
    SELECT e.id INTO id FROM exercises e WHERE e.name = ex_name AND e.deleted_at IS NULL;
    IF id IS NULL THEN
        RAISE EXCEPTION 'seed-trainings: unknown exercise "%"', ex_name;
    END IF;
    RETURN id;
END $$;

CREATE FUNCTION pg_temp.rep_unit(unit_name text) RETURNS uuid LANGUAGE sql AS $$
    SELECT id FROM rep_units WHERE name = unit_name AND deleted_at IS NULL
$$;

CREATE FUNCTION pg_temp.load_unit(unit_name text) RETURNS uuid LANGUAGE sql AS $$
    SELECT id FROM load_units WHERE name = unit_name AND deleted_at IS NULL
$$;

CREATE FUNCTION pg_temp.client(email text, first_name text, last_name text) RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO users (email, first_name, last_name, password_hash)
    VALUES (email, first_name, last_name, crypt('test', gen_salt('bf', 10)))
    RETURNING id
$$;

CREATE FUNCTION pg_temp.training(client uuid, name text, description text, active boolean) RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO trainings (user_id, name, description, active)
    VALUES (client, name, description, active)
    RETURNING id
$$;

CREATE FUNCTION pg_temp.unit(
    training uuid, pos int, name text, weekdays smallint[] DEFAULT '{}', dates date[] DEFAULT '{}', notes text DEFAULT ''
) RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO training_units (training_id, position, name, notes, weekdays, dates)
    VALUES (training, pos, name, notes, weekdays, dates)
    RETURNING id
$$;

-- Sets, reps and load are only set on a block whose kind shares them (sets: not single).
CREATE FUNCTION pg_temp.block(
    unit uuid, pos int, kind text, sets int, name text DEFAULT '', notes text DEFAULT '',
    reps numeric DEFAULT NULL, reps_max numeric DEFAULT NULL, reps_unit text DEFAULT 'reps', per_side boolean DEFAULT false,
    load numeric DEFAULT NULL, load_max numeric DEFAULT NULL, load_unit text DEFAULT 'kg'
) RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO training_blocks (
        unit_id, position, kind, name, sets, notes,
        reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
    ) VALUES (
        unit, pos, kind, name, sets, notes,
        reps, reps_max, CASE WHEN reps IS NOT NULL THEN pg_temp.rep_unit(reps_unit) END, per_side,
        load, load_max, CASE WHEN load IS NOT NULL THEN pg_temp.load_unit(load_unit) END
    )
    RETURNING id
$$;

-- Sets are only set for the exercises of a single block; reps and load stay empty in a complex.
CREATE FUNCTION pg_temp.exercise(
    block uuid, pos int, exercise text, sets int DEFAULT NULL,
    reps numeric DEFAULT NULL, reps_max numeric DEFAULT NULL, reps_unit text DEFAULT 'reps', per_side boolean DEFAULT false,
    load numeric DEFAULT NULL, load_max numeric DEFAULT NULL, load_unit text DEFAULT 'kg', tempo text DEFAULT '',
    notes text DEFAULT ''
) RETURNS void LANGUAGE sql AS $$
    INSERT INTO training_exercises (
        block_id, exercise_id, position, sets, tempo, notes,
        reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
    ) VALUES (
        block, pg_temp.exercise_id(exercise), pos, sets, tempo, notes,
        reps, reps_max, CASE WHEN reps IS NOT NULL THEN pg_temp.rep_unit(reps_unit) END, per_side,
        load, load_max, CASE WHEN load IS NOT NULL THEN pg_temp.load_unit(load_unit) END
    )
$$;

DO $$
DECLARE
    anna uuid;
    ben uuid;
    t uuid;
    u uuid;
    b uuid;
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE email = 'anna@example.com' AND deleted_at IS NULL) THEN
        RETURN;
    END IF;

    anna := pg_temp.client('anna@example.com', 'Anna', 'Muster');
    ben := pg_temp.client('ben@example.com', 'Ben', 'Beispiel');
    PERFORM pg_temp.client('clara@example.com', 'Clara', 'Leer');

    -- An exercise with several videos.
    INSERT INTO exercise_videos (exercise_id, position, url)
    SELECT pg_temp.exercise_id('Squat'), coalesce(max(position), 0) + 1, 'https://www.youtube.com/results?search_query=Squat+tempo+31X1'
    FROM exercise_videos WHERE exercise_id = pg_temp.exercise_id('Squat');

    -- Anna, active: two gym days with every block kind.
    t := pg_temp.training(anna, 'Strength – Phase 2', '4 weeks, heavy compounds. Add 2.5 kg when all reps feel easy.', true);

    u := pg_temp.unit(t, 1, 'Day A', weekdays => '{1,4}');
    b := pg_temp.block(u, 1, 'single', NULL, name => 'Main lift', notes => '3 min rest');
    PERFORM pg_temp.exercise(b, 1, 'Squat', sets => 5, reps => 5, load => 80, load_max => 85, load_unit => '% 1RM', tempo => '31X1');
    PERFORM pg_temp.exercise(b, 2, 'Deadlift', sets => 3, reps => 3, load => 120);
    b := pg_temp.block(u, 2, 'superset', 3, notes => '90s rest after B2');
    PERFORM pg_temp.exercise(b, 1, 'Bench Press', reps => 8, reps_max => 10, load => 60, tempo => '30X1');
    PERFORM pg_temp.exercise(b, 2, 'Barbell Row', reps => 10, load => 50);
    b := pg_temp.block(u, 3, 'superset', 3, name => 'Accessories');
    PERFORM pg_temp.exercise(b, 1, 'Lateral Raise', reps => 12, reps_max => 15, load => 8);
    PERFORM pg_temp.exercise(b, 2, 'Triceps Pushdown', reps => 12, load => 8, load_unit => 'RPE');
    PERFORM pg_temp.exercise(b, 3, 'Calf Raise', reps => 15, notes => 'Pause at the top');

    u := pg_temp.unit(t, 2, 'Day B', weekdays => '{2,5}');
    b := pg_temp.block(u, 1, 'complex', 4, name => 'Barbell complex', notes => 'Don''t put the bar down between movements',
        reps => 3, load => 40);
    PERFORM pg_temp.exercise(b, 1, 'Deadlift');
    PERFORM pg_temp.exercise(b, 2, 'Barbell Row');
    PERFORM pg_temp.exercise(b, 3, 'Overhead Press');
    b := pg_temp.block(u, 2, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Hip Thrust', sets => 4, reps => 8, reps_max => 12, load => 100, tempo => '21X2');
    b := pg_temp.block(u, 3, 'superset', 3);
    PERFORM pg_temp.exercise(b, 1, 'Pull-up', reps => 6, reps_max => 8, load => 2, load_unit => 'RIR');
    PERFORM pg_temp.exercise(b, 2, 'Bicep Curl', reps => 10, load => 12, tempo => '4010', notes => 'Slow eccentric');

    -- Anna, inactive: the previous phase.
    t := pg_temp.training(anna, 'Strength – Phase 1', 'Technique phase, moderate loads.', false);
    u := pg_temp.unit(t, 1, 'Full body', weekdays => '{1,3,5}');
    b := pg_temp.block(u, 1, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Squat', sets => 3, reps => 8, load => 70, load_unit => '% 1RM');
    b := pg_temp.block(u, 2, 'superset', 3);
    PERFORM pg_temp.exercise(b, 1, 'Bench Press', reps => 10, load => 7, load_unit => 'RPE');
    PERFORM pg_temp.exercise(b, 2, 'Barbell Row', reps => 10, load => 7, load_unit => 'RPE');
    b := pg_temp.block(u, 3, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Leg Extension', sets => 3, reps => 12, reps_max => 15);

    -- Ben, active: home training with kettlebells and bodyweight.
    t := pg_temp.training(ben, 'Home functional', 'Kettlebell (16 kg), rings and a box.', true);

    u := pg_temp.unit(t, 1, 'Session 1', weekdays => '{6}', dates => ARRAY[current_date + 3]::date[],
        notes => 'Extra session before the holidays');
    b := pg_temp.block(u, 1, 'single', NULL, name => 'Warm-up');
    PERFORM pg_temp.exercise(b, 1, 'Kettlebell Swing', sets => 2, reps => 20, load => 6, load_unit => 'RPE');
    b := pg_temp.block(u, 2, 'superset', 4, notes => '60s rest after each round');
    PERFORM pg_temp.exercise(b, 1, 'Bulgarian Split Squat', reps => 8, per_side => true, load => 16, tempo => '3110');
    PERFORM pg_temp.exercise(b, 2, 'Ring Row', reps => 10, reps_max => 12);
    b := pg_temp.block(u, 3, 'superset', 3, name => 'Finisher');
    PERFORM pg_temp.exercise(b, 1, 'Walking Lunge', reps => 30, reps_unit => 's');
    PERFORM pg_temp.exercise(b, 2, 'Single Leg Deadlift', reps => 8, per_side => true, load => 16);

    u := pg_temp.unit(t, 2, 'Session 2', weekdays => '{3}');
    b := pg_temp.block(u, 1, 'complex', 3, name => 'Kettlebell complex', reps => 5, load => 16);
    PERFORM pg_temp.exercise(b, 1, 'Kettlebell Swing');
    PERFORM pg_temp.exercise(b, 2, 'Romanian Deadlift');
    PERFORM pg_temp.exercise(b, 3, 'Split Squat');
    b := pg_temp.block(u, 2, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Push-up', sets => 3, reps => 10, reps_max => 15);
    b := pg_temp.block(u, 3, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Box Pistol Squat', sets => 3, reps => 5, per_side => true, notes => 'Lower box if needed');
END $$;
