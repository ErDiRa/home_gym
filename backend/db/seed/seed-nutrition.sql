-- Dev seed: the body and nutrition of me@example.com (needs seed-user.sql; runs after
-- seed-trainings.sql, whose workouts the macros sit next to). Skipped when the user already has a
-- profile.
--   profiles      180 cm, born 1990, male, desk job (activity factor 1.3)
--   body_weights  a weigh-in every few days over the last four weeks, slowly going down, with the
--                 body fat every other time
--   daily_macros  the last 21 days except a few forgotten ones, more carbs on the days with a
--                 finished workout, today's only half logged (protein and carbs)
DO $$
DECLARE
    me UUID := (SELECT id FROM users WHERE email = 'me@example.com' AND deleted_at IS NULL);
BEGIN
    IF EXISTS (SELECT 1 FROM profiles WHERE user_id = me) THEN
        RETURN;
    END IF;

    INSERT INTO profiles (user_id, height_cm, birth_date, sex, activity_factor)
    VALUES (me, 180, '1990-05-14', 'male', 1.3);

    INSERT INTO body_weights (user_id, measured_on, weight_kg, body_fat_percent)
    SELECT me, current_date - d, 82.4 - (27 - d) * 0.04, CASE WHEN d % 6 = 0 THEN 17.8 - (27 - d) * 0.02 END
    FROM generate_series(0, 27, 3) AS d;

    INSERT INTO daily_macros (user_id, day, protein_g, fat_g, carbs_g, notes)
    SELECT me, day,
           150 + (d * 7) % 35,
           65 + (d * 5) % 20,
           CASE WHEN trained THEN 300 + (d * 11) % 60 ELSE 210 + (d * 13) % 50 END,
           CASE WHEN d = 9 THEN 'Dinner out' ELSE '' END
    FROM generate_series(1, 21) AS d,
         LATERAL (SELECT current_date - d AS day) AS days,
         LATERAL (SELECT EXISTS (
             SELECT 1 FROM workouts w
             WHERE w.user_id = me AND w.deleted_at IS NULL AND w.completed_at::date = days.day
         ) AS trained) AS t
    WHERE d NOT IN (4, 12, 17);

    INSERT INTO daily_macros (user_id, day, protein_g, carbs_g)
    VALUES (me, current_date, 95, 140);
END
$$;
