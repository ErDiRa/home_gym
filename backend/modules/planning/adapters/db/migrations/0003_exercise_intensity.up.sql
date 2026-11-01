-- The target intensity of an endurance exercise (an erg, a run): the metric (average watts,
-- calories per hour, or the pace per 500 m or 1000 m in seconds), the target of the work with an
-- optional upper end of a range, and the same for the recovery between intervals.
ALTER TABLE planning.session_exercises
    ADD COLUMN intensity_metric TEXT CHECK (intensity_metric IN ('watts', 'cal_per_hour', 'pace_500m', 'pace_1000m')),
    ADD COLUMN intensity_value  NUMERIC(7, 2),
    ADD COLUMN intensity_max    NUMERIC(7, 2),
    ADD COLUMN recovery_value   NUMERIC(7, 2),
    ADD COLUMN recovery_max     NUMERIC(7, 2),
    ADD CHECK (intensity_metric IS NOT NULL OR (intensity_value IS NULL AND recovery_value IS NULL)),
    ADD CHECK (intensity_max IS NULL OR (intensity_value IS NOT NULL AND intensity_max > intensity_value)),
    ADD CHECK (recovery_max IS NULL OR (recovery_value IS NOT NULL AND recovery_max > recovery_value));
