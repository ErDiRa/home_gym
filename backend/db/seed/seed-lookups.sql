-- Dev seed: joints and muscle groups that exercises pick from.
INSERT INTO joints (name)
SELECT v.name FROM (VALUES ('Ankle'), ('Knee'), ('Hip'), ('Shoulder'), ('Elbow'), ('Wrist'), ('Spine')) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM joints j WHERE j.name = v.name AND j.deleted_at IS NULL);

INSERT INTO muscle_groups (name)
SELECT v.name FROM (VALUES
    ('Chest'), ('Back'), ('Deltoids'), ('Biceps'), ('Triceps'), ('Forearms'),
    ('Quadriceps'), ('Hamstrings'), ('Glutes'), ('Calves'), ('Core')
) AS v(name)
WHERE NOT EXISTS (SELECT 1 FROM muscle_groups m WHERE m.name = v.name AND m.deleted_at IS NULL);
