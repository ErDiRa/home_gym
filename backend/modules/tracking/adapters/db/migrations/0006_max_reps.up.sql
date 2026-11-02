-- A max is for a number of reps: 1 is the one-rep max (what % 1RM loads are calculated from), 3
-- or 5 are rep maxes, from which a 1RM can be estimated. Existing entries are 1RMs.
ALTER TABLE tracking.exercise_maxes
    ADD COLUMN reps INT NOT NULL DEFAULT 1 CHECK (reps BETWEEN 1 AND 30);
