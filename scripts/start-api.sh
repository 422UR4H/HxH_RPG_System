#!/bin/sh
# Applies pending goose migrations, then starts the API.
# The game server never migrates (avoids two processes racing).
set -eu

: "${PG_DB_HOST:?PG_DB_HOST is required}"
: "${PG_DB_USER:?PG_DB_USER is required}"
: "${PG_DB_PASS:?PG_DB_PASS is required}"
: "${PG_DB_NAME:?PG_DB_NAME is required}"

# libpq key=value form avoids URL-encoding problems in passwords.
# Single quotes protect values with spaces; escape \ and ' inside values.
q() { printf "'%s'" "$(printf '%s' "$1" | sed "s/\\\\/\\\\\\\\/g; s/'/\\\\'/g")"; }

DSN="host=$(q "$PG_DB_HOST") port=$(q "${PG_DB_PORT:-5432}") user=$(q "$PG_DB_USER") password=$(q "$PG_DB_PASS") dbname=$(q "$PG_DB_NAME") sslmode=$(q "${PG_DB_SSLMODE:-require}")"

echo "applying migrations to ${PG_DB_HOST}/${PG_DB_NAME}..."
goose -dir ./migrations postgres "$DSN" up

exec ./api
