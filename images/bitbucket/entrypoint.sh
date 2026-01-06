#!/bin/bash
set -euo pipefail

ENTRYPOINT="/usr/bin/tini -- /entrypoint.py --log=INFO"

DB_PATH="/var/atlassian/application-data/bitbucket/shared/data/db.mv.db"
SID_RETRY_COUNT="${SID_RETRY_COUNT:-120}"
WAIT_FOR_DB_UNLOCK_TIMEOUT="${WAIT_FOR_DB_UNLOCK_TIMEOUT:-60}" # seconds to wait for db.lock.db to go away
WAIT_FOR_BOOTSTRAP_STOP="${WAIT_FOR_BOOTSTRAP_STOP:-30}" # seconds to wait for bootstrap to exit after TERM

log() {
	printf '[entrypoint] %s\n' "$*" >&2
}

wait_for_db_unlock() {
	local lock_file="${DB_PATH%.mv.db}.lock.db"
	local elapsed=0
	while [ -e "$lock_file" ]; do
		if [ "$elapsed" -ge "$WAIT_FOR_DB_UNLOCK_TIMEOUT" ]; then
			log "Timed out waiting for H2 lock file $lock_file to disappear (waited ${elapsed}s)"
			exit 1
		fi
		log "H2 lock file present ($lock_file), waiting (t=${elapsed}s)"
		sleep 2
		elapsed=$((elapsed + 2))
	done
	log "H2 lock file cleared ($lock_file)"
}

stop_bootstrap() {
	local pid="$1"
	local elapsed=0
	log "Stopping bootstrap pid=$pid to release DB lock"
	kill -- -"$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
	while kill -0 "$pid" 2>/dev/null; do
		if [ "$elapsed" -ge "$WAIT_FOR_BOOTSTRAP_STOP" ]; then
			log "Bootstrap pid=$pid still running after ${elapsed}s; sending SIGKILL"
			kill -9 -- -"$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
			break
		fi
		log "Waiting for bootstrap pid=$pid to exit (t=${elapsed}s)"
		sleep 2
		elapsed=$((elapsed + 2))
	done
	if wait "$pid" 2>/dev/null; then
		log "Bootstrap stopped cleanly"
	else
		log "Bootstrap wait returned non-zero or already exited"
	fi
}

validate_sid() {
	local sid="$1"
	if [[ ${#sid} -ne 19 ]]; then
		log "SID invalid length ${#sid}: '$sid'"
		return 1
	fi
	if [[ ! $sid =~ ^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$ ]]; then
		log "SID regex mismatch: '$sid'"
		return 1
	fi
	return 0
}

read_server_id() {
	java -cp /opt/atlassian/bitbucket/app/WEB-INF/lib/h2-*.jar org.h2.tools.Shell \
		-url "jdbc:h2:/var/atlassian/application-data/bitbucket/shared/data/db;IFEXISTS=TRUE;ACCESS_MODE_DATA=r" \
		-user sa -password "" \
		-sql "SELECT prop_value FROM app_property WHERE prop_key = 'server.id';" \
		| grep -v "PROP_VALUE" | grep -v "row" | xargs
}

log "Starting entrypoint: DB_PATH=$DB_PATH SID_RETRY_COUNT=$SID_RETRY_COUNT"
BOOT_PID=""

BOOT_LOG=""
if [ ! -f "$DB_PATH" ]; then
	log "DB file missing; starting Bitbucket briefly to bootstrap"
	BOOT_LOG=$(mktemp)
	setsid /entrypoint.py --log=INFO "$@" > "$BOOT_LOG" 2>&1 &
	BOOT_PID=$!
	log "Bootstrap pid=$BOOT_PID"

	log "Watching for 'Generating unique server ID (SID)'"
	if ! ( set +o pipefail; tail -f -n +1 --pid="$BOOT_PID" "$BOOT_LOG" | grep -q "Generating unique server ID (SID)" ); then
		log "Bootstrap process exited without generating SID"
		log "Output:"
		cat "$BOOT_LOG"
		exit 1
	fi
	log "Detected SID generation log line"

	stop_bootstrap "$BOOT_PID"
	rm -f "$BOOT_LOG"

	log "Waiting for H2 lock file to clear"
	wait_for_db_unlock
else
	log "DB file already exists; skipping bootstrap"
fi

SERVER_ID=""
for attempt in $(seq 1 "$SID_RETRY_COUNT"); do
	log "Querying server id (attempt $attempt/$SID_RETRY_COUNT)"
	log "Running: h2 Shell select server.id"
	raw_output=$(read_server_id 2>&1)
	status=$?
	log "Raw server.id status=$status output: '$raw_output'"
	candidate="$raw_output"
	log "SID candidate length=${#candidate}"
	if [ -n "$candidate" ] && validate_sid "$candidate"; then
		SERVER_ID="$candidate"
		log "Captured valid server ID on attempt $attempt"
		break
	fi
	log "Server ID not ready (attempt $attempt/$SID_RETRY_COUNT), retrying..."
	sleep 1
done

if [ -z "$SERVER_ID" ]; then
	log "Failed to capture a valid server ID"
	exit 1
fi

SETUP_LICENSE="$(python3 /license.py "/license_keys/outer_private_key.pem" "/license_keys/inner_private_key.pem" "$SERVER_ID")"
export SETUP_LICENSE
exec $ENTRYPOINT "$@"
