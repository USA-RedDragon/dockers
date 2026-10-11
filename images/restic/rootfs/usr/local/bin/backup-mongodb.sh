#!/usr/bin/env bash
set -u

: "${DB_PASSWORD_FILE:=/mongodb/mongodb-passwords}"

if [ -f "$DB_PASSWORD_FILE" ]; then
  read -r DB_PASSWORD < "$DB_PASSWORD_FILE"
fi

args=(
  --archive
  "--authenticationDatabase=${AUTHENTICATION_DB:-}"
  "--host=$DB_HOST"
  "--username=$DB_USER"
  "--db=$DB_NAME"
)

echo "+ mongodump ${args[*]} --password=***" >&2
set -e
exec mongodump "${args[@]}" --password="$DB_PASSWORD" "$@"
