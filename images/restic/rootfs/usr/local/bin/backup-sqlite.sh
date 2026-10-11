#!/usr/bin/env bash
set -u

db="${1:-$DB_PATH}"

if [[ ! -f "$db" ]]; then
  echo "Database file does not exist: $db"
  exit 1
fi

set -ex
exec sqlite3 -bail "$db" .dump
