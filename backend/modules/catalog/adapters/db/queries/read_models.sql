-- The read models: what the lists and forms show, mapped straight to the HTTP responses.

-- name: ListExercises :many
-- A user's exercises with the names of joints and muscle groups and the focuses as JSON, for lists and the picker.
SELECT
    e.id,
    e.name,
    e.description,
    e.type,
    ARRAY(SELECT url FROM catalog.exercise_videos WHERE exercise_id = e.id ORDER BY position)::text[] AS video_urls,
    e.created_at,
    e.updated_at,
    ARRAY(
        SELECT j.name FROM catalog.exercise_joints ej
        JOIN catalog.joints j ON j.id = ej.joint_id AND j.deleted_at IS NULL
        WHERE ej.exercise_id = e.id ORDER BY j.name
    )::text[] AS joints,
    ARRAY(
        SELECT m.name FROM catalog.exercise_muscle_groups em
        JOIN catalog.muscle_groups m ON m.id = em.muscle_group_id AND m.deleted_at IS NULL
        WHERE em.exercise_id = e.id ORDER BY m.name
    )::text[] AS muscle_groups,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('id', f.id, 'name', f.name, 'alias', f.alias) ORDER BY f.name)
        FROM catalog.exercise_focuses ef
        JOIN catalog.focuses f ON f.id = ef.focus_id AND f.deleted_at IS NULL
        WHERE ef.exercise_id = e.id
    ), '[]'::jsonb)::jsonb AS focuses
FROM catalog.exercises e
WHERE e.user_id = $1 AND e.deleted_at IS NULL
ORDER BY e.name;

-- name: GetExercise :one
-- With the ids of its links, for filling the exercise form.
SELECT
    e.id,
    e.name,
    e.description,
    e.type,
    ARRAY(SELECT url FROM catalog.exercise_videos WHERE exercise_id = e.id ORDER BY position)::text[] AS video_urls,
    ARRAY(SELECT joint_id FROM catalog.exercise_joints WHERE exercise_id = e.id)::uuid[] AS joint_ids,
    ARRAY(SELECT muscle_group_id FROM catalog.exercise_muscle_groups WHERE exercise_id = e.id)::uuid[] AS muscle_group_ids,
    ARRAY(SELECT focus_id FROM catalog.exercise_focuses WHERE exercise_id = e.id)::uuid[] AS focus_ids
FROM catalog.exercises e
WHERE e.id = $1 AND e.user_id = $2 AND e.deleted_at IS NULL;

-- name: ListExerciseRatings :many
-- How a user rated their exercises, for the exercise picker.
SELECT f.exercise_id, f.rating, f.notes, f.updated_at
FROM catalog.exercise_feedbacks f
JOIN catalog.exercises e ON e.id = f.exercise_id AND e.deleted_at IS NULL
WHERE f.user_id = $1 AND f.deleted_at IS NULL;

-- name: ListJoints :many
SELECT id, name FROM catalog.joints
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListMuscleGroups :many
SELECT id, name FROM catalog.muscle_groups
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListFocuses :many
SELECT id, name, alias FROM catalog.focuses
WHERE deleted_at IS NULL
ORDER BY name;

-- name: ListBlockKinds :many
SELECT code, name, description, min_exercises, max_exercises, shared_sets, shared_reps, shared_load, scored
FROM catalog.block_kinds
ORDER BY position;

-- name: ListRepUnits :many
-- Oldest first, so the seeded defaults (reps, s, ...) lead and new units are appended.
SELECT id, name, min_value, max_value FROM catalog.rep_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;

-- name: ListLoadUnits :many
-- Oldest first, like ListRepUnits. kg_per_unit converts between weight units, percent_of_max
-- marks % 1RM.
SELECT id, name, min_value, max_value, kg_per_unit, percent_of_max FROM catalog.load_units
WHERE deleted_at IS NULL
ORDER BY created_at, name;


-- The contract's queries (api/module): what other modules read of the catalog.

-- name: ListExercisesByIDs :many
-- The user's exercises among the ids, deleted ones too (trainings and workouts still show them).
SELECT id, name, type, (deleted_at IS NOT NULL)::boolean AS deleted
FROM catalog.exercises
WHERE user_id = sqlc.arg(user_id) AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: ListAllRepUnits :many
-- Every rep unit, deleted ones too (plans and logged sets still name them).
SELECT id, name, min_value, max_value, (deleted_at IS NOT NULL)::boolean AS deleted
FROM catalog.rep_units
ORDER BY created_at, name;

-- name: ListAllLoadUnits :many
SELECT id, name, min_value, max_value, kg_per_unit, percent_of_max, (deleted_at IS NOT NULL)::boolean AS deleted
FROM catalog.load_units
ORDER BY created_at, name;

-- name: ListExerciseProfiles :many
-- The user's live exercises among the ids with the names of their joints and muscle groups and
-- the user's rating (NULL if unrated), for stats over them.
SELECT
    e.id,
    e.name,
    ARRAY(
        SELECT j.name FROM catalog.exercise_joints ej
        JOIN catalog.joints j ON j.id = ej.joint_id AND j.deleted_at IS NULL
        WHERE ej.exercise_id = e.id ORDER BY j.name
    )::text[] AS joints,
    ARRAY(
        SELECT m.name FROM catalog.exercise_muscle_groups em
        JOIN catalog.muscle_groups m ON m.id = em.muscle_group_id AND m.deleted_at IS NULL
        WHERE em.exercise_id = e.id ORDER BY m.name
    )::text[] AS muscle_groups,
    f.rating
FROM catalog.exercises e
LEFT JOIN catalog.exercise_feedbacks f ON f.exercise_id = e.id AND f.user_id = e.user_id AND f.deleted_at IS NULL
WHERE e.user_id = sqlc.arg(user_id) AND e.id = ANY(sqlc.arg(ids)::uuid[]) AND e.deleted_at IS NULL;
