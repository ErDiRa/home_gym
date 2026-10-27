-- A set done on bodyweight (e.g. pull-ups): bodyweight marks it, load_value/load_unit then hold the
-- optional weight added to it (in a weight unit), or nothing for bodyweight alone.
ALTER TABLE tracking.workout_sets ADD COLUMN bodyweight BOOLEAN NOT NULL DEFAULT false;

-- The log shows such a set's load as "BW" or "BW + 10 kg" (otherwise as in 0002).
CREATE OR REPLACE VIEW tracking.workout_log AS
WITH logged AS (
    SELECT s.workout_id, 'set' AS entry, s.block_position, s.exercise_position, s.set_number,
           e.name AS exercise,
           concat_ws(' ', trim_scale(s.reps_value)::text, ru.name, CASE WHEN s.per_side THEN 'per side' END)
               AS reps,
           -- On bodyweight: "BW", or "BW + 10 kg" with weight added.
           CASE WHEN s.bodyweight
                THEN concat_ws(' + ', 'BW', nullif(concat_ws(' ', trim_scale(s.load_value)::text, lu.name), ''))
                ELSE concat_ws(' ', trim_scale(s.load_value)::text, lu.name)
           END AS load,
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
