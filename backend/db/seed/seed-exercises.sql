-- Dev seed: sample exercises (needs seed-lookups.sql). Skips names that already exist.
-- video_url points to a YouTube search for the exercise; replace with a specific video as needed.
WITH v(name, joints, muscle_groups, description, video_url, focuses) AS (VALUES
    ('Bench Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Triceps', 'Deltoids'], 'Barbell press lying on a flat bench.', 'https://www.youtube.com/results?search_query=Bench+Press+exercise+form', ARRAY['Weightlifting']),
    ('Incline Dumbbell Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Deltoids', 'Triceps'], 'Dumbbell press on a 30° incline bench.', 'https://www.youtube.com/results?search_query=Incline+Dumbbell+Press+exercise+form', ARRAY['Weightlifting']),
    ('Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes', 'Hamstrings'], 'Barbell back squat to parallel.', 'https://www.youtube.com/results?search_query=Squat+exercise+form', ARRAY['Weightlifting', 'Functional fitness']),
    ('Leg Extension', ARRAY['Knee'], ARRAY['Quadriceps'], 'Seated machine knee extension.', 'https://www.youtube.com/results?search_query=Leg+Extension+exercise+form', ARRAY['Knee health']),
    ('Deadlift', ARRAY['Hip', 'Knee'], ARRAY['Hamstrings', 'Glutes', 'Back'], 'Conventional barbell deadlift from the floor.', 'https://www.youtube.com/results?search_query=Deadlift+exercise+form', ARRAY['Weightlifting', 'Functional fitness']),
    ('Hip Thrust', ARRAY['Hip'], ARRAY['Glutes', 'Hamstrings'], 'Barbell hip thrust with shoulders on a bench.', 'https://www.youtube.com/results?search_query=Hip+Thrust+exercise+form', ARRAY['Weightlifting', 'Hip health']),
    ('Overhead Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Deltoids', 'Triceps'], 'Standing barbell press overhead.', 'https://www.youtube.com/results?search_query=Overhead+Press+exercise+form', ARRAY['Weightlifting', 'Shoulder health']),
    ('Lateral Raise', ARRAY['Shoulder'], ARRAY['Deltoids'], 'Dumbbell raise to the side up to shoulder height.', 'https://www.youtube.com/results?search_query=Lateral+Raise+exercise+form', ARRAY['Shoulder health']),
    ('Barbell Row', ARRAY['Elbow', 'Shoulder'], ARRAY['Back', 'Biceps'], 'Bent-over barbell row.', 'https://www.youtube.com/results?search_query=Barbell+Row+exercise+form', ARRAY['Weightlifting']),
    ('Pull-up', ARRAY['Shoulder', 'Elbow'], ARRAY['Back', 'Biceps'], 'Bodyweight pull-up with an overhand grip.', 'https://www.youtube.com/results?search_query=Pull-up+exercise+form', ARRAY['Functional fitness', 'Shoulder health']),
    ('Bicep Curl', ARRAY['Elbow'], ARRAY['Biceps'], 'Standing dumbbell curl.', 'https://www.youtube.com/results?search_query=Bicep+Curl+exercise+form', ARRAY['Weightlifting']),
    ('Triceps Pushdown', ARRAY['Elbow'], ARRAY['Triceps'], 'Cable pushdown with a rope or bar.', 'https://www.youtube.com/results?search_query=Triceps+Pushdown+exercise+form', ARRAY['Weightlifting', 'Elbow health']),
    ('Calf Raise', ARRAY['Ankle'], ARRAY['Calves'], 'Standing calf raise on a step.', 'https://www.youtube.com/results?search_query=Calf+Raise+exercise+form', ARRAY['Functional fitness'])
), new_exercises AS (
    INSERT INTO exercises (name, description, video_url)
    SELECT v.name, v.description, v.video_url
    FROM v
    WHERE NOT EXISTS (
        SELECT 1 FROM exercises e WHERE e.name = v.name AND e.deleted_at IS NULL
    )
    RETURNING id, name
), linked_focuses AS (
    INSERT INTO exercise_focuses (exercise_id, focus_id)
    SELECT n.id, f.id
    FROM new_exercises n
    JOIN v ON v.name = n.name
    CROSS JOIN LATERAL unnest(v.focuses) AS fn(name)
    JOIN focuses f ON lower(f.name) = lower(fn.name) AND f.deleted_at IS NULL
), linked_joints AS (
    INSERT INTO exercise_joints (exercise_id, joint_id)
    SELECT n.id, j.id
    FROM new_exercises n
    JOIN v ON v.name = n.name
    CROSS JOIN LATERAL unnest(v.joints) AS jn(name)
    JOIN joints j ON j.name = jn.name AND j.deleted_at IS NULL
)
INSERT INTO exercise_muscle_groups (exercise_id, muscle_group_id)
SELECT n.id, m.id
FROM new_exercises n
JOIN v ON v.name = n.name
CROSS JOIN LATERAL unnest(v.muscle_groups) AS mn(name)
JOIN muscle_groups m ON m.name = mn.name AND m.deleted_at IS NULL;
