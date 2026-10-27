-- A planned load on bodyweight: load_bodyweight marks an exercise done with the body's weight,
-- e.g. pull-ups, the load columns then hold the optional weight added to it (a belt of 10 kg) in
-- a weight unit, or nothing for bodyweight alone.
ALTER TABLE planning.session_blocks ADD COLUMN load_bodyweight BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE planning.session_exercises ADD COLUMN load_bodyweight BOOLEAN NOT NULL DEFAULT false;
