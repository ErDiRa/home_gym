-- Calories as a load unit: what is logged per set on a machine interval planned by time
-- (e.g. 4 min on the Echo Bike, logged as 52 cal).
INSERT INTO load_units (name, min_value, max_value)
SELECT 'cal', 0, NULL
WHERE NOT EXISTS (SELECT 1 FROM load_units WHERE lower(name) = 'cal' AND deleted_at IS NULL);

-- A planned load may name only its unit, without a value: the unit the sets are logged in.
ALTER TABLE session_blocks DROP CONSTRAINT session_blocks_check2;
ALTER TABLE session_blocks ADD CHECK (load_value IS NULL OR load_unit IS NOT NULL);
ALTER TABLE session_exercises DROP CONSTRAINT session_exercises_check2;
ALTER TABLE session_exercises ADD CHECK (load_value IS NULL OR load_unit IS NOT NULL);
