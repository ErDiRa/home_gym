-- Saved blocks: a user's reusable combinations of exercises (a complex, a superset, a WOD's
-- movements) with the kind of block they make. Only the combination is kept: sets, reps, load and
-- a WOD's format are planned in the session it is added to. Adding one copies it into the
-- session (the client builds the block), so editing either side never changes the other.
CREATE TABLE planning.block_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL REFERENCES catalog.block_kinds (code) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- A name is taken once per user, ignoring case.
CREATE UNIQUE INDEX idx_block_templates_name ON planning.block_templates (user_id, lower(name)) WHERE deleted_at IS NULL;

-- The exercises of a saved block in order (from 1); rewritten on every save.
CREATE TABLE planning.block_template_exercises (
    template_id UUID NOT NULL REFERENCES planning.block_templates (id) ON DELETE CASCADE,
    position    INT NOT NULL CHECK (position >= 1),
    exercise_id UUID NOT NULL REFERENCES catalog.exercises (id) ON DELETE RESTRICT,
    PRIMARY KEY (template_id, position)
);
