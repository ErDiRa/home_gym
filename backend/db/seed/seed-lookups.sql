-- Dev seed: joints, muscle groups and focuses that exercises pick from.
INSERT INTO joints (name)
SELECT v.name FROM (VALUES ('Ankle'), ('Knee'), ('Hip'), ('Shoulder'), ('Elbow'), ('Wrist'), ('Spine')) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM joints j WHERE j.name = v.name AND j.deleted_at IS NULL);

INSERT INTO muscle_groups (name)
SELECT v.name FROM (VALUES
    ('Chest'), ('Back'), ('Deltoids'), ('Biceps'), ('Triceps'), ('Forearms'),
    ('Quadriceps'), ('Hamstrings'), ('Glutes'), ('Calves'), ('Core'), ('Rotator Cuffs')
) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM muscle_groups m WHERE m.name = v.name AND m.deleted_at IS NULL);

INSERT INTO focuses (name, alias)
SELECT v.name, v.alias FROM (VALUES
    ('Weightlifting', 'WL'), ('Functional fitness', 'FUNC'), ('Mobility', 'MOB'),
    ('Shoulder health', 'SHLD'), ('Knee health', 'KNEE'), ('Hip health', 'HIP'), ('Elbow health', 'ELB')
) AS v(name, alias)
WHERE NOT EXISTS (SELECT 1 FROM focuses f WHERE lower(f.name) = lower(v.name) AND f.deleted_at IS NULL);
