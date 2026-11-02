-- name: GetProfile :one
SELECT * FROM nutrition.profiles WHERE user_id = $1;

-- name: SaveProfile :one
INSERT INTO nutrition.profiles (user_id, height_cm, birth_date, sex, activity_factor)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    height_cm = EXCLUDED.height_cm,
    birth_date = EXCLUDED.birth_date,
    sex = EXCLUDED.sex,
    activity_factor = EXCLUDED.activity_factor,
    updated_at = now()
RETURNING *;

-- name: ListBodyWeights :many
-- The user's weights, newest first.
SELECT * FROM nutrition.body_weights
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY measured_on DESC;

-- name: SaveBodyWeight :one
-- A day's weight, replacing the one entered for that day before.
INSERT INTO nutrition.body_weights (user_id, measured_on, weight_kg, body_fat_percent, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, measured_on) WHERE deleted_at IS NULL DO UPDATE SET
    weight_kg = EXCLUDED.weight_kg,
    body_fat_percent = EXCLUDED.body_fat_percent,
    notes = EXCLUDED.notes
RETURNING *;

-- name: DeleteBodyWeight :execrows
UPDATE nutrition.body_weights SET deleted_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListDailyMacros :many
-- The user's macros of the days in [from, to], oldest first.
SELECT * FROM nutrition.daily_macros
WHERE user_id = $1 AND deleted_at IS NULL AND day >= sqlc.arg(from_day) AND day <= sqlc.arg(to_day)
ORDER BY day;

-- name: SaveDailyMacros :one
-- A day's macros, replacing the ones entered for that day before.
INSERT INTO nutrition.daily_macros (user_id, day, protein_g, fat_g, carbs_g, notes)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, day) WHERE deleted_at IS NULL DO UPDATE SET
    protein_g = EXCLUDED.protein_g,
    fat_g = EXCLUDED.fat_g,
    carbs_g = EXCLUDED.carbs_g,
    notes = EXCLUDED.notes,
    updated_at = now()
RETURNING *;

-- name: DeleteDailyMacros :execrows
UPDATE nutrition.daily_macros SET deleted_at = now()
WHERE user_id = $1 AND day = $2 AND deleted_at IS NULL;
