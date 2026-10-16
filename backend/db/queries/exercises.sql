-- name: CreateExercise :one
WITH new_exercise AS (
    INSERT INTO exercises (name, description, video_url)
    VALUES ($1, $2, $3)
    RETURNING *
), linked_joints AS (
    INSERT INTO exercise_joints (exercise_id, joint_id)
    SELECT id, unnest(sqlc.arg(joint_ids)::uuid[]) FROM new_exercise
), linked_muscle_groups AS (
    INSERT INTO exercise_muscle_groups (exercise_id, muscle_group_id)
    SELECT id, unnest(sqlc.arg(muscle_group_ids)::uuid[]) FROM new_exercise
), linked_focuses AS (
    INSERT INTO exercise_focuses (exercise_id, focus_id)
    SELECT id, unnest(sqlc.arg(focus_ids)::uuid[]) FROM new_exercise
)
SELECT id, name, description, video_url, created_at, updated_at FROM new_exercise;

-- name: ListExercises :many
SELECT
    e.id,
    e.name,
    e.description,
    e.video_url,
    e.created_at,
    e.updated_at,
    ARRAY(
        SELECT j.name FROM exercise_joints ej
        JOIN joints j ON j.id = ej.joint_id AND j.deleted_at IS NULL
        WHERE ej.exercise_id = e.id ORDER BY j.name
    )::text[] AS joints,
    ARRAY(
        SELECT m.name FROM exercise_muscle_groups em
        JOIN muscle_groups m ON m.id = em.muscle_group_id AND m.deleted_at IS NULL
        WHERE em.exercise_id = e.id ORDER BY m.name
    )::text[] AS muscle_groups,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('id', f.id, 'name', f.name, 'alias', f.alias) ORDER BY f.name)
        FROM exercise_focuses ef
        JOIN focuses f ON f.id = ef.focus_id AND f.deleted_at IS NULL
        WHERE ef.exercise_id = e.id
    ), '[]'::jsonb) AS focuses
FROM exercises e
WHERE e.deleted_at IS NULL
ORDER BY e.name;

-- name: GetExercise :one
SELECT
    e.id,
    e.name,
    e.description,
    e.video_url,
    ARRAY(SELECT joint_id FROM exercise_joints WHERE exercise_id = e.id)::uuid[] AS joint_ids,
    ARRAY(SELECT muscle_group_id FROM exercise_muscle_groups WHERE exercise_id = e.id)::uuid[] AS muscle_group_ids,
    ARRAY(SELECT focus_id FROM exercise_focuses WHERE exercise_id = e.id)::uuid[] AS focus_ids
FROM exercises e
WHERE e.id = $1 AND e.deleted_at IS NULL;

-- name: UpdateExercise :one
-- The links are synced by deleting the ones no longer wanted and inserting the new ones;
-- the two never touch the same row, so they are safe in one statement.
WITH updated AS (
    UPDATE exercises
    SET name = $2, description = $3, video_url = $4, updated_at = now()
    WHERE id = $1 AND deleted_at IS NULL
    RETURNING *
), removed_joints AS (
    DELETE FROM exercise_joints
    WHERE exercise_id IN (SELECT id FROM updated) AND joint_id <> ALL(sqlc.arg(joint_ids)::uuid[])
), added_joints AS (
    INSERT INTO exercise_joints (exercise_id, joint_id)
    SELECT id, unnest(sqlc.arg(joint_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
), removed_muscle_groups AS (
    DELETE FROM exercise_muscle_groups
    WHERE exercise_id IN (SELECT id FROM updated) AND muscle_group_id <> ALL(sqlc.arg(muscle_group_ids)::uuid[])
), added_muscle_groups AS (
    INSERT INTO exercise_muscle_groups (exercise_id, muscle_group_id)
    SELECT id, unnest(sqlc.arg(muscle_group_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
), removed_focuses AS (
    DELETE FROM exercise_focuses
    WHERE exercise_id IN (SELECT id FROM updated) AND focus_id <> ALL(sqlc.arg(focus_ids)::uuid[])
), added_focuses AS (
    INSERT INTO exercise_focuses (exercise_id, focus_id)
    SELECT id, unnest(sqlc.arg(focus_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
)
SELECT id, name, description, video_url, created_at, updated_at FROM updated;

-- name: ListJoints :many
SELECT id, name FROM joints
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListFocuses :many
SELECT id, name, alias FROM focuses
WHERE deleted_at IS NULL
ORDER BY name;

-- name: CreateFocus :one
INSERT INTO focuses (name, alias) VALUES ($1, $2)
RETURNING id, name, alias;

-- name: ListMuscleGroups :many
SELECT id, name FROM muscle_groups
WHERE deleted_at IS NULL
ORDER BY name;

-- name: UpsertExerciseFeedback :one
INSERT INTO exercise_feedbacks (user_id, exercise_id, rating, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, exercise_id) WHERE deleted_at IS NULL
DO UPDATE SET rating = EXCLUDED.rating,
              notes = EXCLUDED.notes,
              updated_at = now()
RETURNING *;
