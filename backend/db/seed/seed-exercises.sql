-- Dev seed: sample exercises (needs seed-lookups.sql and seed-user.sql), owned by me@example.com.
-- Skips names the user already has.
-- video_url is a YouTube form tutorial for the exercise (a /watch link, so the app embeds it).
WITH v(name, joints, muscle_groups, description, video_url, focuses, type) AS (VALUES
    ('Bench Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Triceps', 'Deltoids'], 'Barbell press lying on a flat bench.', 'https://www.youtube.com/watch?v=4Y2ZdHCOXok', ARRAY['Weightlifting'], 'strength'),
    ('Incline Dumbbell Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Deltoids', 'Triceps'], 'Dumbbell press on a 30° incline bench.', 'https://www.youtube.com/watch?v=IP4oeKh1Sd4', ARRAY['Weightlifting'], 'strength'),
    ('Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes', 'Hamstrings'], 'Barbell back squat to parallel.', 'https://www.youtube.com/watch?v=ultWZbUMPL8', ARRAY['Weightlifting', 'Functional fitness'], 'strength'),
    ('Leg Extension', ARRAY['Knee'], ARRAY['Quadriceps'], 'Seated machine knee extension.', 'https://www.youtube.com/watch?v=ljO4jkwv8wQ', ARRAY['Knee health'], 'strength'),
    ('Deadlift', ARRAY['Hip', 'Knee'], ARRAY['Hamstrings', 'Glutes', 'Back'], 'Conventional barbell deadlift from the floor.', 'https://www.youtube.com/watch?v=XxWcirHIwVo', ARRAY['Weightlifting', 'Functional fitness'], 'strength'),
    ('Hip Thrust', ARRAY['Hip'], ARRAY['Glutes', 'Hamstrings'], 'Barbell hip thrust with shoulders on a bench.', 'https://www.youtube.com/watch?v=xDmFkJxPzeM', ARRAY['Weightlifting', 'Hip health'], 'strength'),
    ('Overhead Press', ARRAY['Shoulder', 'Elbow'], ARRAY['Deltoids', 'Triceps'], 'Standing barbell press overhead.', 'https://www.youtube.com/watch?v=QAQ64hK4Xxs', ARRAY['Weightlifting', 'Shoulder health'], 'strength'),
    ('Lateral Raise', ARRAY['Shoulder'], ARRAY['Deltoids'], 'Dumbbell raise to the side up to shoulder height.', 'https://www.youtube.com/watch?v=PzsMitRdI_8', ARRAY['Shoulder health'], 'strength'),
    ('Barbell Row', ARRAY['Elbow', 'Shoulder'], ARRAY['Back', 'Biceps'], 'Bent-over barbell row.', 'https://www.youtube.com/watch?v=kBWAon7ItDw', ARRAY['Weightlifting'], 'strength'),
    ('Pull-up', ARRAY['Shoulder', 'Elbow'], ARRAY['Back', 'Biceps'], 'Bodyweight pull-up with an overhand grip.', 'https://www.youtube.com/watch?v=eGo4IYlbE5g', ARRAY['Functional fitness', 'Shoulder health'], 'strength'),
    ('Bicep Curl', ARRAY['Elbow'], ARRAY['Biceps'], 'Standing dumbbell curl.', 'https://www.youtube.com/watch?v=cBSD6mQIPQk', ARRAY['Weightlifting'], 'strength'),
    ('Triceps Pushdown', ARRAY['Elbow'], ARRAY['Triceps'], 'Cable pushdown with a rope or bar.', 'https://www.youtube.com/watch?v=-zLyUAo1gMw', ARRAY['Weightlifting', 'Elbow health'], 'strength'),
    ('Calf Raise', ARRAY['Ankle'], ARRAY['Calves'], 'Standing calf raise on a step.', 'https://www.youtube.com/watch?v=SVtg-1loH4c', ARRAY['Functional fitness'], 'strength'),
    ('Kettlebell Swing', ARRAY['Hip'], ARRAY['Glutes', 'Hamstrings', 'Core'], 'Two-handed hip hinge swing to chest height.', 'https://www.youtube.com/watch?v=hp3qVqIHNOI', ARRAY['Functional fitness'], 'strength'),
    ('Romanian Deadlift', ARRAY['Hip'], ARRAY['Hamstrings', 'Glutes'], 'Hip hinge with soft knees, weight close to the legs.', 'https://www.youtube.com/watch?v=_oyxCn2iSjU', ARRAY['Functional fitness', 'Hip health'], 'strength'),
    ('Single Leg Deadlift', ARRAY['Hip'], ARRAY['Hamstrings', 'Glutes'], 'Hip hinge on one leg, the free leg reaching back.', 'https://www.youtube.com/watch?v=Zfr6wizR8rs', ARRAY['Functional fitness', 'Hip health'], 'strength'),
    ('Bulgarian Split Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Split squat with the rear foot on a bench.', 'https://www.youtube.com/watch?v=SkNsa3eBwLA', ARRAY['Functional fitness', 'Knee health'], 'strength'),
    ('Split Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Static lunge, lowering the back knee towards the floor.', 'https://www.youtube.com/watch?v=la0pLPq-3A8', ARRAY['Functional fitness'], 'strength'),
    ('Walking Lunge', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Alternating forward lunges while walking.', 'https://www.youtube.com/watch?v=DlhojghkaQ0', ARRAY['Functional fitness'], 'strength'),
    ('Box Pistol Squat', ARRAY['Knee', 'Hip'], ARRAY['Quadriceps', 'Glutes'], 'Single-leg squat down to a box.', 'https://www.youtube.com/watch?v=kc4YfSHjbYE', ARRAY['Functional fitness', 'Knee health'], 'strength'),
    ('Push-up', ARRAY['Shoulder', 'Elbow'], ARRAY['Chest', 'Triceps'], 'Bodyweight push-up, hands elevated to make it easier.', 'https://www.youtube.com/watch?v=IODxDxX7oi4', ARRAY['Functional fitness'], 'strength'),
    ('Ring Row', ARRAY['Shoulder', 'Elbow'], ARRAY['Back', 'Biceps'], 'Inverted row on gymnastic rings.', 'https://www.youtube.com/watch?v=_SsRDWz3U6g', ARRAY['Functional fitness', 'Shoulder health'], 'strength')
), new_exercises AS (
    INSERT INTO exercises (user_id, name, description, type)
    SELECT u.id, v.name, v.description, v.type
    FROM v
    JOIN users u ON u.email = 'me@example.com' AND u.deleted_at IS NULL
    WHERE NOT EXISTS (
        SELECT 1 FROM exercises e WHERE e.user_id = u.id AND e.name = v.name AND e.deleted_at IS NULL
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
