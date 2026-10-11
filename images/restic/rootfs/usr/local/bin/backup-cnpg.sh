#!/usr/bin/env bash
set -u

: "${CNPG_SECRET_PATH:=/postgresql-app}"

read -r DB_HOST < "$CNPG_SECRET_PATH/host"
read -r DB_NAME < "$CNPG_SECRET_PATH/dbname"
read -r DB_USER < "$CNPG_SECRET_PATH/username"
read -r PGPASSWORD < "$CNPG_SECRET_PATH/password"
export PGPASSWORD

if [[ -z "${PG_RESTRICT_KEY:-}" ]]; then
  PG_RESTRICT_KEY="$(sha256sum <<<"$DB_HOST$DB_NAME$DB_USER$PGPASSWORD" | cut -d' ' -f1)"
fi

set -ex
exec pg_dump --clean --if-exists --no-owner --restrict-key="$PG_RESTRICT_KEY" \
  --host="$DB_HOST" --username="$DB_USER" --dbname="$DB_NAME" "$@"
