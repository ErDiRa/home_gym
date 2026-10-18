-- name: CreateExercise :one
-- Inserts the exercise and its links in one statement: Postgres runs every data-modifying CTE,
-- even the ones nothing selects from. Video positions come from the array order.
WITH new_exercise AS (
    INSERT INTO exercises (user_id, name, description, type)
    VALUES ($1, $2, $3, $4)
    RETURNING *
), linked_videos AS (
    INSERT INTO exercise_videos (exercise_id, position, url)
    SELECT n.id, v.position, v.url
    FROM new_exercise n, unnest(sqlc.arg(video_urls)::text[]) WITH ORDINALITY AS v(url, position)
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
SELECT id, name, description, type, sqlc.arg(video_urls)::text[] AS video_urls, created_at, updated_at FROM new_exercise;

-- name: ListExercises :many
-- A user's exercises with the names of joints and muscle groups and the focuses as JSON, for lists and the picker.
SELECT
    e.id,
    e.name,
    e.description,
    e.type,
    ARRAY(SELECT url FROM exercise_videos WHERE exercise_id = e.id ORDER BY position)::text[] AS video_urls,
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
WHERE e.user_id = $1 AND e.deleted_at IS NULL
ORDER BY e.name;

-- name: GetExercise :one
-- With the ids of its links, for filling the exercise form.
SELECT
    e.id,
    e.name,
    e.description,
    e.type,
    ARRAY(SELECT url FROM exercise_videos WHERE exercise_id = e.id ORDER BY position)::text[] AS video_urls,
    ARRAY(SELECT joint_id FROM exercise_joints WHERE exercise_id = e.id)::uuid[] AS joint_ids,
    ARRAY(SELECT muscle_group_id FROM exercise_muscle_groups WHERE exercise_id = e.id)::uuid[] AS muscle_group_ids,
    ARRAY(SELECT focus_id FROM exercise_focuses WHERE exercise_id = e.id)::uuid[] AS focus_ids
FROM exercises e
WHERE e.id = $1 AND e.user_id = $2 AND e.deleted_at IS NULL;

-- name: UpdateExercise :one
-- The links are synced by deleting the ones no longer wanted and inserting the new ones;
-- the two never touch the same row, so they are safe in one statement. Videos are ordered,
-- so they are all replaced. Another user's exercise updates nothing (no row).
WITH updated AS (
    UPDATE exercises
    SET name = $3, description = $4, type = $5, updated_at = now()
    WHERE exercises.id = $1 AND exercises.user_id = $2 AND exercises.deleted_at IS NULL
    RETURNING *
), removed_videos AS (
    DELETE FROM exercise_videos ev USING updated u WHERE ev.exercise_id = u.id
), added_videos AS (
    INSERT INTO exercise_videos (exercise_id, position, url)
    SELECT u.id, v.position, v.url
    FROM updated u, unnest(sqlc.arg(video_urls)::text[]) WITH ORDINALITY AS v(url, position)
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
SELECT id, name, description, type, sqlc.arg(video_urls)::text[] AS video_urls, created_at, updated_at FROM updated;
