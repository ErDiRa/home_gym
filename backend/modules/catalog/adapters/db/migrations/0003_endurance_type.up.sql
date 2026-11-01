-- A third exercise type: endurance (running, rowing, the bike...), next to mobility and strength.
ALTER TABLE catalog.exercises DROP CONSTRAINT exercises_type_check;
ALTER TABLE catalog.exercises ADD CONSTRAINT exercises_type_check CHECK (type IN ('mobility', 'strength', 'endurance'));
