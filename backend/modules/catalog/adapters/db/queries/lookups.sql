-- name: CreateFocus :one
INSERT INTO catalog.focuses (name, alias) VALUES ($1, $2)
RETURNING id, name, alias;

-- name: CreateRepUnit :one
INSERT INTO catalog.rep_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value;

-- name: CreateLoadUnit :one
INSERT INTO catalog.load_units (name, min_value, max_value) VALUES ($1, $2, $3)
RETURNING id, name, min_value, max_value, kg_per_unit, percent_of_max;
