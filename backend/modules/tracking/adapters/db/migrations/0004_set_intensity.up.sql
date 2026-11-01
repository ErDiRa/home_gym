-- The average intensity of a set of an endurance exercise (an erg, a run), as the machine showed
-- it: watts, calories per hour, or the pace per 500 m or 1000 m in seconds.
ALTER TABLE tracking.workout_sets
    ADD COLUMN intensity_metric TEXT CHECK (intensity_metric IN ('watts', 'cal_per_hour', 'pace_500m', 'pace_1000m')),
    ADD COLUMN intensity_value  NUMERIC(7, 2) CHECK (intensity_value > 0),
    ADD CHECK ((intensity_metric IS NULL) = (intensity_value IS NULL));

-- The log shows it after the load (otherwise as in 0003).
CREATE OR REPLACE VIEW tracking.workout_log AS
WITH logged AS (
    SELECT s.workout_id, 'set' AS entry, s.block_position, s.exercise_position, s.set_number,
           e.name AS exercise,
           concat_ws(' ', trim_scale(s.reps_value)::text, ru.name, CASE WHEN s.per_side THEN 'per side' END)
               AS reps,
           -- On bodyweight: "BW", or "BW + 10 kg" with weight added.
           -- The average intensity follows the load, e.g. "42 cal · 1:55 /500 m".
           concat_ws(' · ',
               nullif(CASE WHEN s.bodyweight
                           THEN concat_ws(' + ', 'BW', nullif(concat_ws(' ', trim_scale(s.load_value)::text, lu.name), ''))
                           ELSE concat_ws(' ', trim_scale(s.load_value)::text, lu.name)
                      END, ''),
               CASE s.intensity_metric
                   WHEN 'watts' THEN trim_scale(s.intensity_value)::text || ' W'
                   WHEN 'cal_per_hour' THEN trim_scale(s.intensity_value)::text || ' cal/h'
                   WHEN 'pace_500m' THEN to_char(make_interval(secs => s.intensity_value), 'FMMI:SS') || ' /500 m'
                   WHEN 'pace_1000m' THEN to_char(make_interval(secs => s.intensity_value), 'FMMI:SS') || ' /1000 m'
               END) AS load,
           NULL::text AS score, NULL::boolean AS rx, NULL::text AS notes,
           max(s.exercise_position) OVER (PARTITION BY s.workout_id, s.block_position) > 1 AS numbered,
           s.completed_at AS logged_at, s.id AS set_id, NULL::uuid AS result_id, s.exercise_id
    FROM tracking.workout_sets s
    JOIN catalog.exercises e ON e.id = s.exercise_id
    JOIN catalog.rep_units ru ON ru.id = s.reps_unit
    LEFT JOIN catalog.load_units lu ON lu.id = s.load_unit
    UNION ALL
    SELECT r.workout_id, 'score', r.block_position, NULL, NULL,
           NULL, NULL, NULL,
           r.score_type || ': ' || concat_ws(' + ',
               to_char(make_interval(secs => r.time_seconds),
                       CASE WHEN r.time_seconds >= 3600 THEN 'FMHH24:MI:SS' ELSE 'FMMI:SS' END),
               r.rounds || ' rounds', r.reps || ' reps', r.calories || ' cal'),
           r.rx, nullif(r.notes, ''),
           false, r.completed_at, NULL, r.id, NULL
    FROM tracking.workout_results r
)
SELECT u.email                                    AS user_email,
       w.completed_at::date                       AS done_on,
       t.name                                     AS training,
       coalesce(ws.week, (w.started_at::date - t.start_date) / 7 + 1) AS week,
       w.session_name                             AS session,
       chr(64 + l.block_position) || CASE WHEN l.numbered THEN l.exercise_position::text ELSE '' END
                                                  AS label,
       l.entry,
       l.exercise,
       l.set_number                               AS set,
       nullif(l.reps, '')                         AS reps,
       nullif(l.load, '')                         AS load,
       l.score,
       l.rx,
       l.notes,
       w.notes                                    AS workout_notes,
       w.started_at,
       w.completed_at                             AS done_at,
       l.logged_at,
       w.user_id, w.training_id, w.session_id, w.week_session_id, w.id AS workout_id,
       l.set_id, l.result_id, l.exercise_id
FROM logged l
JOIN tracking.workouts w ON w.id = l.workout_id AND w.deleted_at IS NULL
JOIN identity.users u ON u.id = w.user_id
LEFT JOIN planning.trainings t ON t.id = w.training_id
LEFT JOIN planning.training_week_sessions ws ON ws.id = w.week_session_id
ORDER BY coalesce(w.completed_at, w.started_at) DESC, w.id, l.block_position, l.exercise_position NULLS FIRST, l.set_number;
