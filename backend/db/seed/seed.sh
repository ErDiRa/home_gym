#!/bin/sh
# Dev seed, run inside the db container once the backend has migrated the database (task
# backend:fresh): the user me@example.com / test, the sample exercises, every *.sql in
# db/seed/local (private seed data that stays out of git, e.g. seed-exercise-archive.sql; skipped
# when empty) and the sample trainings, in that order. The seeds name tables without their module
# schema, so the search path lists the schemas (public for pgcrypto's crypt()).
set -e
export PGOPTIONS="-c search_path=identity,catalog,planning,tracking,whoop,public"
run() {
    echo "seed: $1"
    psql -q -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$1"
}
run /seed/seed-user.sql
run /seed/seed-exercises.sql
for f in /seed/local/*.sql; do
    [ -e "$f" ] && run "$f"
done
run /seed/seed-trainings.sql
