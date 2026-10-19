-- Dev seed: a session library, sample trainings and workouts of me@example.com (needs the
-- schema's block kinds and units, seed-user.sql and seed-exercises.sql). Skipped when the user
-- already has a training. Squat also gets a second video link.
--   Library             session templates the trainings are copied from
--   Strength – Phase 2  active, 4 weeks from this week's Monday, 2-3 sessions a week (one
--                       repeated), all block kinds, ranges, kg / % 1RM / RPE / RIR loads; the
--                       first session of this week done today
--   CrossFit warm-ups   active too, alongside: a warm-up per CrossFit day, 4-5 a week
--   Strength – Phase 1  the previous 4 weeks, with finished workouts and their sets
--   Home functional     the next phase, inactive: kettlebell work, seconds, per-side reps
-- The user also rates some exercises (1-5), for the rating filter in the exercise picker.

-- Helpers, dropped with the database session. Units and exercises are looked up by name.
CREATE FUNCTION pg_temp.me() RETURNS uuid LANGUAGE sql AS $$
    SELECT id FROM users WHERE email = 'me@example.com' AND deleted_at IS NULL
$$;

CREATE FUNCTION pg_temp.exercise_id(ex_name text) RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE id uuid;
BEGIN
    SELECT e.id INTO id FROM exercises e WHERE e.user_id = pg_temp.me() AND e.name = ex_name AND e.deleted_at IS NULL;
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

-- The period is given in weeks from its first day; this_monday() is the start of the current week.
CREATE FUNCTION pg_temp.this_monday() RETURNS date LANGUAGE sql AS $$
    SELECT date_trunc('week', current_date)::date
$$;

CREATE FUNCTION pg_temp.training(name text, description text, active boolean, start_date date, weeks int)
RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO trainings (user_id, name, description, active, start_date, weeks)
    VALUES (pg_temp.me(), name, description, active, start_date, weeks)
    RETURNING id
$$;

-- A session template of the library.
CREATE FUNCTION pg_temp.template(name text, notes text DEFAULT '')
RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO sessions (user_id, name, notes)
    VALUES (pg_temp.me(), name, notes)
    RETURNING id
$$;

-- Adds a template to a training the way the app does: as a copy with its blocks and exercises,
-- optionally under another name.
CREATE FUNCTION pg_temp.copy_session(template uuid, training uuid, pos int, new_name text DEFAULT NULL)
RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE
    new_session uuid;
    src record;
    block uuid;
BEGIN
    INSERT INTO sessions (user_id, training_id, position, name, notes)
    SELECT user_id, training, pos, coalesce(new_name, name), notes FROM sessions WHERE id = template
    RETURNING id INTO new_session;
    FOR src IN SELECT * FROM session_blocks WHERE session_id = template ORDER BY position LOOP
        INSERT INTO session_blocks (
            session_id, position, kind, name, sets, notes,
            reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
        ) VALUES (
            new_session, src.position, src.kind, src.name, src.sets, src.notes,
            src.reps_value, src.reps_max, src.reps_unit, src.per_side, src.load_value, src.load_max, src.load_unit
        ) RETURNING id INTO block;
        INSERT INTO session_exercises (
            block_id, exercise_id, position, sets, tempo, notes,
            reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
        )
        SELECT block, exercise_id, position, sets, tempo, notes,
               reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
        FROM session_exercises WHERE block_id = src.id;
    END LOOP;
    RETURN new_session;
END $$;

-- Plans a week of a training: the sessions in order (repeats allowed).
CREATE FUNCTION pg_temp.plan(training uuid, week int, sessions uuid[])
RETURNS void LANGUAGE sql AS $$
    INSERT INTO training_week_sessions (training_id, week, position, session_id)
    SELECT training, week, s.position, s.session_id
    FROM unnest(sessions) WITH ORDINALITY AS s(session_id, position)
$$;

-- The slot at a position of a week's plan.
CREATE FUNCTION pg_temp.slot(training uuid, week int, pos int)
RETURNS uuid LANGUAGE sql AS $$
    SELECT id FROM training_week_sessions ws
    WHERE ws.training_id = training AND ws.week = slot.week AND ws.position = pos
$$;

-- Sets, reps and load are only set on a block whose kind shares them (sets: not single).
CREATE FUNCTION pg_temp.block(
    session uuid, pos int, kind text, sets int, name text DEFAULT '', notes text DEFAULT '',
    reps numeric DEFAULT NULL, reps_max numeric DEFAULT NULL, reps_unit text DEFAULT 'reps', per_side boolean DEFAULT false,
    load numeric DEFAULT NULL, load_max numeric DEFAULT NULL, load_unit text DEFAULT 'kg'
) RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO session_blocks (
        session_id, position, kind, name, sets, notes,
        reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
    ) VALUES (
        session, pos, kind, name, sets, notes,
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
    INSERT INTO session_exercises (
        block_id, exercise_id, position, sets, tempo, notes,
        reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit
    ) VALUES (
        block, pg_temp.exercise_id(exercise), pos, sets, tempo, notes,
        reps, reps_max, CASE WHEN reps IS NOT NULL THEN pg_temp.rep_unit(reps_unit) END, per_side,
        load, load_max, CASE WHEN load IS NOT NULL THEN pg_temp.load_unit(load_unit) END
    )
$$;

-- A finished workout of a planned slot, done at the given time and taking an hour.
CREATE FUNCTION pg_temp.workout(slot uuid, done_at timestamptz, notes text DEFAULT '')
RETURNS uuid LANGUAGE sql AS $$
    INSERT INTO workouts (user_id, training_id, week_session_id, session_id, session_name, started_at, completed_at, notes)
    SELECT pg_temp.me(), ws.training_id, ws.id, s.id, s.name, done_at - interval '1 hour', done_at, notes
    FROM training_week_sessions ws JOIN sessions s ON s.id = ws.session_id
    WHERE ws.id = slot
    RETURNING id
$$;

-- Logs sets 1..sets of an exercise at its place in the session, all with the same reps and load.
CREATE FUNCTION pg_temp.sets(
    workout uuid, block int, pos int, exercise text, sets int, reps numeric, reps_unit text DEFAULT 'reps',
    load numeric DEFAULT NULL, load_unit text DEFAULT 'kg', per_side boolean DEFAULT false
) RETURNS void LANGUAGE sql AS $$
    INSERT INTO workout_sets (
        workout_id, exercise_id, block_position, exercise_position, set_number,
        reps_value, reps_unit, per_side, load_value, load_unit, completed_at
    )
    SELECT workout, pg_temp.exercise_id(exercise), block, pos, n,
           reps, pg_temp.rep_unit(reps_unit), per_side,
           load, CASE WHEN load IS NOT NULL THEN pg_temp.load_unit(load_unit) END,
           w.started_at + n * interval '3 minutes'
    FROM generate_series(1, sets) AS n, workouts w
    WHERE w.id = workout
$$;

DO $$
DECLARE
    t uuid; -- the training, session, block and workout being filled
    u uuid;
    s1 uuid;
    s2 uuid;
    b uuid;
    w uuid;
    gym_1 uuid; -- library templates
    gym_2 uuid;
    technique uuid;
    kb_1 uuid;
    kb_2 uuid;
    warm_squat uuid;
    warm_pull uuid;
    week int;
    session int;
BEGIN
    IF pg_temp.me() IS NULL OR EXISTS (SELECT 1 FROM trainings WHERE user_id = pg_temp.me()) THEN
        RETURN;
    END IF;

    -- An exercise with several videos.
    INSERT INTO exercise_videos (exercise_id, position, url)
    SELECT pg_temp.exercise_id('Squat'), coalesce(max(position), 0) + 1, 'https://www.youtube.com/results?search_query=Squat+tempo+31X1'
    FROM exercise_videos WHERE exercise_id = pg_temp.exercise_id('Squat');

    -- Exercise ratings: some exercises are rated, the rest not.
    INSERT INTO exercise_feedbacks (user_id, exercise_id, rating, notes)
    SELECT pg_temp.me(), pg_temp.exercise_id(r.exercise), r.rating, r.notes
    FROM (VALUES
        ('Squat', 5, 'Favourite lift'),
        ('Deadlift', 4, ''),
        ('Bench Press', 3, 'Shoulder feels it at the bottom'),
        ('Lateral Raise', 2, ''),
        ('Triceps Pushdown', 1, 'Boring'),
        ('Pull-up', 5, ''),
        ('Kettlebell Swing', 5, ''),
        ('Push-up', 4, ''),
        ('Bulgarian Split Squat', 2, 'Knee hurts on the back leg'),
        ('Ring Row', 3, '')
    ) AS r(exercise, rating, notes);

    -- The library: session templates with every block kind.
    u := pg_temp.template('Gym – squat and bench');
    gym_1 := u;
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

    u := pg_temp.template('Gym – complex and pull');
    gym_2 := u;
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

    u := pg_temp.template('Technique full body');
    technique := u;
    b := pg_temp.block(u, 1, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Squat', sets => 3, reps => 8, load => 70, load_unit => '% 1RM');
    b := pg_temp.block(u, 2, 'superset', 3);
    PERFORM pg_temp.exercise(b, 1, 'Bench Press', reps => 10, load => 7, load_unit => 'RPE');
    PERFORM pg_temp.exercise(b, 2, 'Barbell Row', reps => 10, load => 7, load_unit => 'RPE');
    b := pg_temp.block(u, 3, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Leg Extension', sets => 3, reps => 12, reps_max => 15);

    u := pg_temp.template('Kettlebell – legs and rows', notes => 'Short on time? Do the first two blocks.');
    kb_1 := u;
    b := pg_temp.block(u, 1, 'single', NULL, name => 'Warm-up');
    PERFORM pg_temp.exercise(b, 1, 'Kettlebell Swing', sets => 2, reps => 20, load => 6, load_unit => 'RPE');
    b := pg_temp.block(u, 2, 'superset', 4, notes => '60s rest after each round');
    PERFORM pg_temp.exercise(b, 1, 'Bulgarian Split Squat', reps => 8, per_side => true, load => 16, tempo => '3110');
    PERFORM pg_temp.exercise(b, 2, 'Ring Row', reps => 10, reps_max => 12);
    b := pg_temp.block(u, 3, 'superset', 3, name => 'Finisher');
    PERFORM pg_temp.exercise(b, 1, 'Walking Lunge', reps => 30, reps_unit => 's');
    PERFORM pg_temp.exercise(b, 2, 'Single Leg Deadlift', reps => 8, per_side => true, load => 16);

    u := pg_temp.template('Kettlebell – complex and push');
    kb_2 := u;
    b := pg_temp.block(u, 1, 'complex', 3, name => 'Kettlebell complex', reps => 5, load => 16);
    PERFORM pg_temp.exercise(b, 1, 'Kettlebell Swing');
    PERFORM pg_temp.exercise(b, 2, 'Romanian Deadlift');
    PERFORM pg_temp.exercise(b, 3, 'Split Squat');
    b := pg_temp.block(u, 2, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Push-up', sets => 3, reps => 10, reps_max => 15);
    b := pg_temp.block(u, 3, 'single', NULL);
    PERFORM pg_temp.exercise(b, 1, 'Box Pistol Squat', sets => 3, reps => 5, per_side => true, notes => 'Lower box if needed');

    u := pg_temp.template('Warm-up – squat day', notes => 'Before the CrossFit class; light and quick.');
    warm_squat := u;
    b := pg_temp.block(u, 1, 'superset', 2);
    PERFORM pg_temp.exercise(b, 1, 'Walking Lunge', reps => 30, reps_unit => 's');
    PERFORM pg_temp.exercise(b, 2, 'Split Squat', reps => 8, per_side => true);
    PERFORM pg_temp.exercise(b, 3, 'Kettlebell Swing', reps => 10, load => 5, load_unit => 'RPE');

    u := pg_temp.template('Warm-up – pull day', notes => 'Before the CrossFit class; light and quick.');
    warm_pull := u;
    b := pg_temp.block(u, 1, 'superset', 2);
    PERFORM pg_temp.exercise(b, 1, 'Ring Row', reps => 10);
    PERFORM pg_temp.exercise(b, 2, 'Push-up', reps => 8);
    PERFORM pg_temp.exercise(b, 3, 'Kettlebell Swing', reps => 10, load => 5, load_unit => 'RPE');

    -- Active alongside the strength block: warm-ups for the CrossFit days of the week.
    t := pg_temp.training('CrossFit warm-ups', 'One per CrossFit day, planned when the week''s programming is out.', true,
        pg_temp.this_monday(), 4);
    s1 := pg_temp.copy_session(warm_squat, t, 1);
    s2 := pg_temp.copy_session(warm_pull, t, 2);
    PERFORM pg_temp.plan(t, 1, ARRAY[s1, s2, s1, s2, s1]);
    PERFORM pg_temp.plan(t, 2, ARRAY[s2, s1, s2, s1]);
    PERFORM pg_temp.plan(t, 3, ARRAY[s1, s2, s1, s2, s1]);
    PERFORM pg_temp.plan(t, 4, ARRAY[s2, s1, s2]);

    -- Active: two sessions from the library, three a week with one repeated, two in the
    -- lighter last week.
    t := pg_temp.training('Strength – Phase 2', '4 weeks, heavy compounds. Add 2.5 kg when all reps feel easy.', true,
        pg_temp.this_monday(), 4);
    s1 := pg_temp.copy_session(gym_1, t, 1, 'Session 1');
    s2 := pg_temp.copy_session(gym_2, t, 2, 'Session 2');
    PERFORM pg_temp.plan(t, 1, ARRAY[s1, s2, s1]);
    PERFORM pg_temp.plan(t, 2, ARRAY[s2, s1, s2]);
    PERFORM pg_temp.plan(t, 3, ARRAY[s1, s2, s1]);
    PERFORM pg_temp.plan(t, 4, ARRAY[s1, s2]);

    -- The first session of this week done today.
    w := pg_temp.workout(pg_temp.slot(t, 1, 1), now() - interval '2 hours', 'Squats moved fast.');
    PERFORM pg_temp.sets(w, 1, 1, 'Squat', 5, 5, load => 100);
    PERFORM pg_temp.sets(w, 1, 2, 'Deadlift', 3, 3, load => 120);
    PERFORM pg_temp.sets(w, 2, 1, 'Bench Press', 3, 9, load => 60);
    PERFORM pg_temp.sets(w, 2, 2, 'Barbell Row', 3, 10, load => 50);

    -- Inactive: the previous phase, one session three times a week, with its workouts getting
    -- heavier.
    t := pg_temp.training('Strength – Phase 1', 'Technique phase, moderate loads.', false,
        pg_temp.this_monday() - 28, 4);
    s1 := pg_temp.copy_session(technique, t, 1, 'Session 1');
    FOR week IN 1..4 LOOP
        PERFORM pg_temp.plan(t, week, ARRAY[s1, s1, s1]);
        FOR session IN 1..3 LOOP
            CONTINUE WHEN week = 3 AND session = 3; -- a missed session
            w := pg_temp.workout(pg_temp.slot(t, week, session),
                (pg_temp.this_monday() - 35 + week * 7 + (session - 1) * 2)::timestamptz + interval '18 hours');
            PERFORM pg_temp.sets(w, 1, 1, 'Squat', 3, 8, load => 77.5 + week * 2.5);
            PERFORM pg_temp.sets(w, 2, 1, 'Bench Press', 3, 10, load => 47.5 + week * 2.5);
            PERFORM pg_temp.sets(w, 2, 2, 'Barbell Row', 3, 10, load => 37.5 + week * 2.5);
            PERFORM pg_temp.sets(w, 3, 1, 'Leg Extension', 3, 11 + week);
        END LOOP;
    END LOOP;

    -- Inactive: the next phase, home training with kettlebells and bodyweight; a deload week
    -- with one session every fourth week.
    t := pg_temp.training('Home functional', 'Kettlebell (16 kg), rings and a box.', false,
        pg_temp.this_monday() + 28, 8);
    s1 := pg_temp.copy_session(kb_1, t, 1, 'Session 1');
    s2 := pg_temp.copy_session(kb_2, t, 2, 'Session 2');
    FOR week IN 1..8 LOOP
        IF week % 4 = 0 THEN
            PERFORM pg_temp.plan(t, week, ARRAY[s1]);
        ELSE
            PERFORM pg_temp.plan(t, week, ARRAY[s1, s2]);
        END IF;
    END LOOP;
END $$;
