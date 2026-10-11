#!/usr/bin/env bash
set -u

: "${DB_PASSWORD_FILE:=/mariadb/mariadb-password}"

if [[ -f "$DB_PASSWORD_FILE" ]]; then
  read -r DB_PASSWORD < "$DB_PASSWORD_FILE"
fi

export MYSQL_PWD="$DB_PASSWORD"

set -ex
exec mariadb-dump --add-drop-table --skip-dump-date --single-transaction \
  --host="$DB_HOST" --user="$DB_USER" "$DB_NAME" "$@"
