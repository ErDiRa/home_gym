#!/bin/sh
# Dev seed: runs every *.sql in db/seed/local (mounted at /seed-local), in name order.
# That folder holds private seed data that stays out of git, e.g. seed-exercise-archive.sql;
# when it is empty, nothing happens.
set -e
for f in /seed-local/*.sql; do
    [ -e "$f" ] || continue
    echo "seed-local: running $f"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$f"
done
