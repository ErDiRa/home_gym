-- The lists exercises pick from (joints, muscle groups, focuses), shared by all users. Skips
-- names already there, so a database that has them (e.g. from the former seed) is unaffected.
INSERT INTO catalog.joints (name)
SELECT v.name FROM (VALUES ('Ankle'), ('Knee'), ('Hip'), ('Shoulder'), ('Elbow'), ('Wrist'), ('Spine')) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM catalog.joints j WHERE j.name = v.name AND j.deleted_at IS NULL);

INSERT INTO catalog.muscle_groups (name)
SELECT v.name FROM (VALUES
    ('Chest'), ('Back'), ('Deltoids'), ('Biceps'), ('Triceps'), ('Forearms'),
    ('Quadriceps'), ('Hamstrings'), ('Glutes'), ('Calves'), ('Core'), ('Rotator Cuffs')
) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM catalog.muscle_groups m WHERE m.name = v.name AND m.deleted_at IS NULL);

INSERT INTO catalog.focuses (name, alias)
SELECT v.name, v.alias FROM (VALUES
    ('Weightlifting', 'WL'), ('Functional fitness', 'FUNC'), ('Mobility', 'MOB'),
    ('Shoulder health', 'SHLD'), ('Knee health', 'KNEE'), ('Hip health', 'HIP'), ('Elbow health', 'ELB')
) AS v(name, alias)
WHERE NOT EXISTS (SELECT 1 FROM catalog.focuses f WHERE lower(f.name) = lower(v.name) AND f.deleted_at IS NULL);
