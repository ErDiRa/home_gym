-- Writing and loading a training with its whole tree (TrainingRepository).

-- name: GetTraining :one
-- Only the user's own trainings: another user's is not found.
SELECT id, cycle_id, name, description, active, start_date, end_date, weeks, created_at, updated_at
FROM planning.trainings
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListTrainingSessions :many
SELECT id, name, notes FROM planning.sessions
WHERE training_id = sqlc.arg(training_id)::uuid
ORDER BY position;

-- name: ListTrainingBlocks :many
SELECT sb.id, sb.session_id, sb.kind, sb.name, sb.sets,
       sb.reps_value, sb.reps_max, sb.reps_unit, sb.per_side, sb.load_value, sb.load_max, sb.load_unit,
       sb.score_type, sb.time_seconds, sb.interval_seconds, sb.notes
FROM planning.session_blocks sb
JOIN planning.sessions s ON s.id = sb.session_id
WHERE s.training_id = sqlc.arg(training_id)::uuid
ORDER BY sb.session_id, sb.position;

-- name: ListTrainingExercises :many
SELECT se.block_id, se.exercise_id, se.sets,
       se.reps_value, se.reps_max, se.reps_unit, se.per_side, se.load_value, se.load_max, se.load_unit,
       se.tempo, se.notes
FROM planning.session_exercises se
JOIN planning.session_blocks sb ON sb.id = se.block_id
JOIN planning.sessions s ON s.id = sb.session_id
WHERE s.training_id = sqlc.arg(training_id)::uuid
ORDER BY se.block_id, se.position;

-- name: ListWeekSessions :many
-- The week plan of a training: its slots by week, in order.
SELECT id, week, position, session_id FROM planning.training_week_sessions
WHERE training_id = $1
ORDER BY week, position;

-- name: CreateTraining :one
-- The cycle is checked to be the user's before (UserOwnsCycle).
INSERT INTO planning.trainings (user_id, cycle_id, name, description, active, start_date, weeks)
VALUES (sqlc.arg(user_id), sqlc.narg(cycle_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(active),
        sqlc.arg(start_date), sqlc.arg(weeks))
RETURNING id, cycle_id, name, description, active, start_date, end_date, weeks, created_at, updated_at;

-- name: UpdateTraining :one
-- Another user's training updates nothing (no row).
UPDATE planning.trainings
SET cycle_id = sqlc.narg(cycle_id), name = sqlc.arg(name), description = sqlc.arg(description),
    active = sqlc.arg(active), start_date = sqlc.arg(start_date), weeks = sqlc.arg(weeks), updated_at = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL
RETURNING id, cycle_id, name, description, active, start_date, end_date, weeks, created_at, updated_at;

-- name: DeleteTraining :execrows
UPDATE planning.trainings SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: InsertSession :one
-- A training's session (training_id set) or a library template (training_id NULL).
INSERT INTO planning.sessions (user_id, training_id, position, name, notes)
VALUES (sqlc.arg(user_id), sqlc.narg(training_id), sqlc.arg(position), sqlc.arg(name), sqlc.arg(notes))
RETURNING id;

-- name: UpdateTrainingSession :execrows
-- Matching training_id too means a session id from another training updates nothing (0 rows).
UPDATE planning.sessions
SET position = sqlc.arg(position), name = sqlc.arg(name), notes = sqlc.arg(notes), updated_at = now()
WHERE id = sqlc.arg(id) AND training_id = sqlc.arg(training_id)::uuid;

-- name: DeleteTrainingSessionsExcept :exec
-- Removes the sessions dropped from the training (their slots go with them).
DELETE FROM planning.sessions
WHERE training_id = sqlc.arg(training_id)::uuid AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: DeleteTrainingBlocks :exec
-- Clears all blocks of a training before they are re-inserted; exercises go with them (cascade).
DELETE FROM planning.session_blocks sb
USING planning.sessions s
WHERE s.id = sb.session_id AND s.training_id = sqlc.arg(training_id)::uuid;

-- name: InsertWeekSession :exec
INSERT INTO planning.training_week_sessions (training_id, week, position, session_id)
VALUES ($1, $2, $3, $4);

-- name: UpdateWeekSession :execrows
-- Matching training_id too means a slot id from another training updates nothing (0 rows).
UPDATE planning.training_week_sessions
SET week = $3, position = $4, session_id = $5
WHERE id = $1 AND training_id = $2;

-- name: DeleteWeekSessionsExcept :exec
-- Removes the slots dropped from the week plan.
DELETE FROM planning.training_week_sessions
WHERE training_id = $1 AND id <> ALL(sqlc.arg(keep_ids)::uuid[]);

-- name: InsertSessionBlock :one
-- sqlc.narg marks the parameters that may be NULL (what the block's kind doesn't share).
INSERT INTO planning.session_blocks (
    session_id, position, kind, name, sets,
    reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit,
    score_type, time_seconds, interval_seconds, notes
) VALUES (
    sqlc.arg(session_id), sqlc.arg(position), sqlc.arg(kind), sqlc.arg(name), sqlc.narg(sets),
    sqlc.narg(reps_value), sqlc.narg(reps_max), sqlc.narg(reps_unit), sqlc.arg(per_side),
    sqlc.narg(load_value), sqlc.narg(load_max), sqlc.narg(load_unit),
    sqlc.narg(score_type), sqlc.narg(time_seconds), sqlc.narg(interval_seconds), sqlc.arg(notes)
)
RETURNING id;

-- name: InsertSessionExercise :exec
-- The exercise is checked to be one of the user's live ones before (the catalog's contract).
INSERT INTO planning.session_exercises (
    block_id, exercise_id, position, sets,
    reps_value, reps_max, reps_unit, per_side, load_value, load_max, load_unit, tempo, notes
) VALUES (
    sqlc.arg(block_id), sqlc.arg(exercise_id), sqlc.arg(position), sqlc.narg(sets),
    sqlc.narg(reps_value), sqlc.narg(reps_max), sqlc.narg(reps_unit), sqlc.arg(per_side),
    sqlc.narg(load_value), sqlc.narg(load_max), sqlc.narg(load_unit), sqlc.arg(tempo), sqlc.arg(notes)
);

-- name: UserOwnsCycle :one
-- Whether the cycle is one of the user's live cycles, before a training is put into it.
SELECT EXISTS (
    SELECT 1 FROM planning.cycles WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
);
