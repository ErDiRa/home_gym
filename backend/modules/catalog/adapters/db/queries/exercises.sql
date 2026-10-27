-- name: CreateExercise :one
-- Inserts the exercise and its links in one statement: Postgres runs every data-modifying CTE,
-- even the ones nothing selects from. Video positions come from the array order.
WITH new_exercise AS (
    INSERT INTO catalog.exercises (user_id, name, description, type)
    VALUES ($1, $2, $3, $4)
    RETURNING *
), linked_videos AS (
    INSERT INTO catalog.exercise_videos (exercise_id, position, url)
    SELECT n.id, v.position, v.url
    FROM new_exercise n, unnest(sqlc.arg(video_urls)::text[]) WITH ORDINALITY AS v(url, position)
), linked_joints AS (
    INSERT INTO catalog.exercise_joints (exercise_id, joint_id)
    SELECT id, unnest(sqlc.arg(joint_ids)::uuid[]) FROM new_exercise
), linked_muscle_groups AS (
    INSERT INTO catalog.exercise_muscle_groups (exercise_id, muscle_group_id)
    SELECT id, unnest(sqlc.arg(muscle_group_ids)::uuid[]) FROM new_exercise
), linked_focuses AS (
    INSERT INTO catalog.exercise_focuses (exercise_id, focus_id)
    SELECT id, unnest(sqlc.arg(focus_ids)::uuid[]) FROM new_exercise
)
SELECT id, name, description, type, sqlc.arg(video_urls)::text[] AS video_urls, created_at, updated_at FROM new_exercise;

-- name: UpdateExercise :one
-- The links are synced by deleting the ones no longer wanted and inserting the new ones;
-- the two never touch the same row, so they are safe in one statement. Videos are ordered,
-- so they are all replaced. Another user's exercise updates nothing (no row).
WITH updated AS (
    UPDATE catalog.exercises
    SET name = $3, description = $4, type = $5, updated_at = now()
    WHERE exercises.id = $1 AND exercises.user_id = $2 AND exercises.deleted_at IS NULL
    RETURNING *
), removed_videos AS (
    DELETE FROM catalog.exercise_videos ev USING updated u WHERE ev.exercise_id = u.id
), added_videos AS (
    INSERT INTO catalog.exercise_videos (exercise_id, position, url)
    SELECT u.id, v.position, v.url
    FROM updated u, unnest(sqlc.arg(video_urls)::text[]) WITH ORDINALITY AS v(url, position)
), removed_joints AS (
    DELETE FROM catalog.exercise_joints
    WHERE exercise_id IN (SELECT id FROM updated) AND joint_id <> ALL(sqlc.arg(joint_ids)::uuid[])
), added_joints AS (
    INSERT INTO catalog.exercise_joints (exercise_id, joint_id)
    SELECT id, unnest(sqlc.arg(joint_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
), removed_muscle_groups AS (
    DELETE FROM catalog.exercise_muscle_groups
    WHERE exercise_id IN (SELECT id FROM updated) AND muscle_group_id <> ALL(sqlc.arg(muscle_group_ids)::uuid[])
), added_muscle_groups AS (
    INSERT INTO catalog.exercise_muscle_groups (exercise_id, muscle_group_id)
    SELECT id, unnest(sqlc.arg(muscle_group_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
), removed_focuses AS (
    DELETE FROM catalog.exercise_focuses
    WHERE exercise_id IN (SELECT id FROM updated) AND focus_id <> ALL(sqlc.arg(focus_ids)::uuid[])
), added_focuses AS (
    INSERT INTO catalog.exercise_focuses (exercise_id, focus_id)
    SELECT id, unnest(sqlc.arg(focus_ids)::uuid[]) FROM updated
    ON CONFLICT DO NOTHING
)
SELECT id, name, description, type, sqlc.arg(video_urls)::text[] AS video_urls, created_at, updated_at FROM updated;
