-- Dev seed: sample exercises (needs seed-lookups.sql). Skips names that already exist.
-- video_url points to a YouTube search for the exercise; replace with a specific video as needed.
WITH v(name, joints, muscle_groups, description, video_url, focuses, type) AS (VALUES
    ('Bench Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Triceps', 'Deltoids'], 'Barbell press lying on a flat bench.', 'https://www.youtube.com/results?search_query=Bench+Press+exercise+form', ARRAY['Weightlifting'], 'strength'),
    ('Incline Dumbbell Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Deltoids', 'Triceps'], 'Dumbbell press on a 30° incline bench.', 'https://www.youtube.com/results?search_query=Incline+Dumbbell+Press+exercise+form', ARRAY['Weightlifting'], 'strength'),
    ('Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes', 'Hamstrings'], 'Barbell back squat to parallel.', 'https://www.youtube.com/results?search_query=Squat+exercise+form', ARRAY['Weightlifting', 'Functional fitness'], 'strength'),
    ('Leg Extension', ARRAY['Knee'], ARRAY['Quadriceps'], 'Seated machine knee extension.', 'https://www.youtube.com/results?search_query=Leg+Extension+exercise+form', ARRAY['Knee health'], 'strength'),
    ('Deadlift', ARRAY['Hip', 'Knee'], ARRAY['Hamstrings', 'Glutes', 'Back'], 'Conventional barbell deadlift from the floor.', 'https://www.youtube.com/results?search_query=Deadlift+exercise+form', ARRAY['Weightlifting', 'Functional fitness'], 'strength'),
    ('Hip Thrust', ARRAY['Hip'], ARRAY['Glutes', 'Hamstrings'], 'Barbell hip thrust with shoulders on a bench.', 'https://www.youtube.com/results?search_query=Hip+Thrust+exercise+form', ARRAY['Weightlifting', 'Hip health'], 'strength'),
    ('Overhead Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Deltoids', 'Triceps'], 'Standing barbell press overhead.', 'https://www.youtube.com/results?search_query=Overhead+Press+exercise+form', ARRAY['Weightlifting', 'Shoulder health'], 'strength'),
    ('Lateral Raise', ARRAY['Shoulder'], ARRAY['Deltoids'], 'Dumbbell raise to the side up to shoulder height.', 'https://www.youtube.com/results?search_query=Lateral+Raise+exercise+form', ARRAY['Shoulder health'], 'strength'),
    ('Barbell Row', ARRAY['Elbow', 'Shoulder'], ARRAY['Back', 'Biceps'], 'Bent-over barbell row.', 'https://www.youtube.com/results?search_query=Barbell+Row+exercise+form', ARRAY['Weightlifting'], 'strength'),
    ('Pull-up', ARRAY['Shoulder', 'Elbow'], ARRAY['Back', 'Biceps'], 'Bodyweight pull-up with an overhand grip.', 'https://www.youtube.com/results?search_query=Pull-up+exercise+form', ARRAY['Functional fitness', 'Shoulder health'], 'strength'),
    ('Bicep Curl', ARRAY['Elbow'], ARRAY['Biceps'], 'Standing dumbbell curl.', 'https://www.youtube.com/results?search_query=Bicep+Curl+exercise+form', ARRAY['Weightlifting'], 'strength'),
    ('Triceps Pushdown', ARRAY['Elbow'], ARRAY['Triceps'], 'Cable pushdown with a rope or bar.', 'https://www.youtube.com/results?search_query=Triceps+Pushdown+exercise+form', ARRAY['Weightlifting', 'Elbow health'], 'strength'),
    ('Calf Raise', ARRAY['Ankle'], ARRAY['Calves'], 'Standing calf raise on a step.', 'https://www.youtube.com/results?search_query=Calf+Raise+exercise+form', ARRAY['Functional fitness'], 'strength'),
    ('Kettlebell Swing', ARRAY['Hip'], ARRAY['Glutes', 'Hamstrings', 'Core'], 'Two-handed hip hinge swing to chest height.', 'https://www.youtube.com/results?search_query=Kettlebell+Swing+exercise+form', ARRAY['Functional fitness'], 'strength'),
    ('Romanian Deadlift', ARRAY['Hip'], ARRAY['Hamstrings', 'Glutes'], 'Hip hinge with soft knees, weight close to the legs.', 'https://www.youtube.com/results?search_query=Romanian+Deadlift+exercise+form', ARRAY['Functional fitness', 'Hip health'], 'strength'),
    ('Single Leg Deadlift', ARRAY['Hip'], ARRAY['Hamstrings', 'Glutes'], 'Hip hinge on one leg, the free leg reaching back.', 'https://www.youtube.com/results?search_query=Single+Leg+Deadlift+exercise+form', ARRAY['Functional fitness', 'Hip health'], 'strength'),
    ('Bulgarian Split Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Split squat with the rear foot on a bench.', 'https://www.youtube.com/results?search_query=Bulgarian+Split+Squat+exercise+form', ARRAY['Functional fitness', 'Knee health'], 'strength'),
    ('Split Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Static lunge, lowering the back knee towards the floor.', 'https://www.youtube.com/results?search_query=Split+Squat+exercise+form', ARRAY['Functional fitness'], 'strength'),
    ('Walking Lunge', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Alternating forward lunges while walking.', 'https://www.youtube.com/results?search_query=Walking+Lunge+exercise+form', ARRAY['Functional fitness'], 'strength'),
    ('Box Pistol Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Single-leg squat down to a box.', 'https://www.youtube.com/results?search_query=Box+Pistol+Squat+exercise+form', ARRAY['Functional fitness', 'Knee health'], 'strength'),
    ('Push-up', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Triceps'], 'Bodyweight push-up, hands elevated to make it easier.', 'https://www.youtube.com/results?search_query=Push-up+exercise+form', ARRAY['Functional fitness'], 'strength'),
    ('Ring Row', ARRAY['Shoulder', 'Elbow'], ARRAY['Back', 'Biceps'], 'Inverted row on gymnastic rings.', 'https://www.youtube.com/results?search_query=Ring+Row+exercise+form', ARRAY['Functional fitness', 'Shoulder health'], 'strength')
), new_exercises AS (
    INSERT INTO exercises (name, description, type)
    SELECT v.name, v.description, v.type
    FROM v
    WHERE NOT EXISTS (
        SELECT 1 FROM exercises e WHERE e.name = v.name AND e.deleted_at IS NULL
    )
    RETURNING id, name -- the link CTEs join back to v by name for the rest of the row
), linked_videos AS (
    INSERT INTO exercise_videos (exercise_id, position, url)
    SELECT n.id, 1, v.video_url
    FROM new_exercises n
    JOIN v ON v.name = n.name
    WHERE v.video_url <> ''
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
