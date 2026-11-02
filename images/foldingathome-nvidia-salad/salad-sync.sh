#!/bin/bash
# Keeps the client's state in S3 so a replacement node can resume the same units.
# Usage: salad-sync restore | salad-sync push | salad-sync watch
# restore exits non-zero when the saved state cannot be read; the caller must then not start the client.
# push returns 2 when another container has taken over this state.
set -u
DATA=/var/lib/fah-client
SNAP=/tmp/fah-snap
RESTORED=/tmp/fah-restored
OWNER=/tmp/fah-owner
REMOTE="s3:${FAH_S3_BUCKET:-fah-salad}/${FAH_S3_PREFIX:-default}"
EXCLUDE=(--exclude 'viewerFrame*.json' --exclude 'viewerTop.json' --exclude 'logfile_*.txt')

export RCLONE_CONFIG_S3_TYPE=s3 RCLONE_CONFIG_S3_PROVIDER=Minio RCLONE_CONFIG_S3_ENDPOINT="${FAH_S3_URL:-}"
export RCLONE_CONFIG_S3_ACCESS_KEY_ID="${FAH_S3_ACCESS_KEY:-}" RCLONE_CONFIG_S3_SECRET_ACCESS_KEY="${FAH_S3_SECRET_KEY:-}"
export RCLONE_S3_NO_CHECK_BUCKET=true RCLONE_LOG_LEVEL=${RCLONE_LOG_LEVEL:-ERROR}
export RCLONE_CONTIMEOUT=15s RCLONE_TIMEOUT=120s RCLONE_RETRIES=3 RCLONE_LOW_LEVEL_RETRIES=3

log() { echo "salad-sync: $*" >&2; }

enabled() { [ -n "${FAH_S3_URL:-}" ] && [ -n "${FAH_S3_ACCESS_KEY:-}" ] && [ -n "${FAH_S3_SECRET_KEY:-}" ]; }

# Lists the top of the prefix; fails (instead of returning nothing) when S3 cannot be reached.
list_top() {
  local try out
  for try in 1 2 3 4 5 6; do
    if out=$(rclone lsf "$REMOTE" --max-depth 1 2>/dev/null); then printf '%s\n' "$out"; return 0; fi
    log "cannot list $REMOTE (try $try)"
    sleep $((try * 10))
  done
  return 1
}

# Prints the slot named by "current", or nothing when there is none yet. Fails when it cannot tell.
read_current() {
  local top cur
  top=$(list_top) || return 1
  grep -qx current <<<"$top" || return 0
  cur=$(rclone cat "$REMOTE/current" 2>/dev/null | tr -cd 'ab') || return 1
  [ -n "$cur" ] || return 1
  echo "$cur"
}

claim() {
  cat /proc/sys/kernel/random/uuid > "$OWNER"
  rclone rcat "$REMOTE/owner" < "$OWNER" || { log "cannot write owner token"; return 1; }
}

# Saves alternate between slot-a and slot-b and "current" names the last complete one, so a node killed mid-upload never leaves a mixed checkpoint behind.
restore() {
  rm -f "$RESTORED"
  enabled || { log "no S3 settings, starting fresh"; touch "$RESTORED"; return 0; }
  if [ -f "$DATA/client.db" ]; then
    log "local state present (restarted in place), keeping it"
    claim || return 1
    touch "$RESTORED"; return 0
  fi
  local top cur src try
  top=$(list_top) || { log "S3 unreachable, not starting"; return 1; }
  if grep -qx current <<<"$top"; then
    cur=$(read_current) && [ -n "$cur" ] || { log "cannot read $REMOTE/current, not starting"; return 1; }
    src="$REMOTE/slot-$cur"
  elif grep -qx client.db <<<"$top"; then
    src="$REMOTE"
  elif grep -qE '^slot-' <<<"$top"; then
    log "slots exist but no current pointer at $REMOTE, not starting"; return 1
  else
    log "nothing saved at $REMOTE, starting fresh"
    claim || return 1
    touch "$RESTORED"; return 0
  fi
  for try in 1 2 3; do
    rm -rf "$DATA/work" "$DATA"/client.db*
    if rclone copy "$src" "$DATA" "${EXCLUDE[@]}" --exclude '/slot-*/**' --exclude '/current' --exclude '/owner' --transfers 8; then
      claim || return 1
      touch "$RESTORED"
      log "restored $(find "$DATA/work" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l) unit(s) and client.db from $src"
      return 0
    fi
    log "restore from $src failed (try $try)"
    sleep $((try * 10))
  done
  rm -rf "$DATA/work" "$DATA"/client.db*
  log "restore failed, not starting"
  return 1
}

tree_stamp() { find "$DATA/client.db" "$DATA/client.db-wal" "$DATA/work" -type f ! -name '*.log' ! -name '*.xvg' ! -name 'viewer*' ! -name 'logfile_*' -printf '%p %s %T@\n' 2>/dev/null | sort; }

# Copies the client state to local disk first: the cores append to logs and rewrite checkpoints while rclone uploads, and S3 rejects a file that grows mid-PUT.
snapshot() {
  local f ok
  for _ in 1 2 3 4 5; do
    before=$(tree_stamp)
    rm -rf "$SNAP" && mkdir -p "$SNAP/work" || return 1
    ok=1
    cp -a "$DATA/work/." "$SNAP/work/" || ok=0
    for f in client.db client.db-wal; do [ ! -f "$DATA/$f" ] || cp "$DATA/$f" "$SNAP/$f" || ok=0; done
    [ "$ok" = 1 ] && [ "$before" = "$(tree_stamp)" ] && return 0
    sleep 2
  done
  return 1
}

# Drops units that belong to an earlier client identity (rows and folders) from the snapshot, so they are not carried
# from node to node. Skips when the client ID is unknown or no saved unit belongs to it.
prune() {
  local id own u dropped=0
  [ -f /tmp/fah-id ] || return 0
  id=$(tr -cd 'A-Za-z0-9_-' < /tmp/fah-id)
  [ -n "$id" ] && [ -f "$SNAP/client.db" ] || return 0
  own="$(printf %s "$id" | tr '_-' '/+')="
  [ "$(sqlite3 "$SNAP/client.db" "SELECT count(*) FROM units WHERE json_extract(value, '\$.data.request.data.id') = '$own'")" -gt 0 ] || return 0
  sqlite3 "$SNAP/client.db" "DELETE FROM units WHERE json_extract(value, '\$.data.request.data.id') IS NOT NULL AND json_extract(value, '\$.data.request.data.id') != '$own'" || return 0
  local keep; keep=$(sqlite3 "$SNAP/client.db" "SELECT name FROM units") || return 0
  for d in "$SNAP/work"/*/; do
    [ -d "$d" ] || continue
    u=$(basename "$d")
    grep -qxF -- "$u" <<<"$keep" || { rm -rf "$d"; dropped=$((dropped + 1)); }
  done
  if [ "$dropped" -gt 0 ] && [ "$dropped" != "$(cat /tmp/fah-pruned 2>/dev/null)" ]; then
    echo "$dropped" > /tmp/fah-pruned
    log "not saving $dropped unit folder(s) of earlier identities"
  fi
}

# One save at a time: the watcher, a final save on SIGTERM and manual saves share the same snapshot folder.
push() {
  enabled || return 0
  [ -f "$RESTORED" ] || { log "save skipped, restore did not complete"; return 1; }
  # The lock lives only in this subshell, so it is released when the save ends.
  (
    flock -w 300 9 || { log "save skipped, another save is still running"; exit 1; }
    push_locked
  ) 9>/tmp/fah-push.lock
}

push_locked() {
  local owner cur next
  owner=$(rclone cat "$REMOTE/owner" 2>/dev/null) || { log "save failed, cannot read owner token"; return 1; }
  if [ "$owner" != "$(cat "$OWNER" 2>/dev/null)" ]; then
    log "another container now owns $REMOTE, stopping saves"
    return 2
  fi
  mkdir -p "$DATA/work"
  snapshot || { log "save skipped, could not copy a steady state"; return 1; }
  prune
  cur=$(read_current) || { log "save failed, cannot read $REMOTE/current"; return 1; }
  next=a
  [ "$cur" = a ] && next=b
  # Files unchanged since the last save are copied from the other slot inside S3 instead of crossing the node's uplink again.
  local reuse=()
  [ -n "$cur" ] && reuse=(--copy-dest "$REMOTE/slot-$cur")
  if rclone sync "$SNAP" "$REMOTE/slot-$next" "${EXCLUDE[@]}" "${reuse[@]}" --transfers 4 && echo "$next" | rclone rcat "$REMOTE/current"; then
    log "saved $(date -u +%H:%M:%S) to slot-$next"
    if [ -n "$(rclone lsf "$REMOTE/client.db" 2>/dev/null)" ]; then
      rclone purge "$REMOTE/work" 2>/dev/null; rclone delete "$REMOTE" --max-depth 1 --include 'client.db*' 2>/dev/null
    fi
  else
    log "save failed"
    return 1
  fi
}

watch() {
  enabled || { log "no S3 settings, not saving state"; exec sleep infinity; }
  mkdir -p "$DATA/work"
  local last=0 ev urgent rc until
  # A unit resumes from its last checkpoint, so saves follow checkpoints instead of a short timer: a growing trajectory
  # is re-uploaded whole on every save, and on slow node uplinks frequent saves delayed the checkpoint that matters.
  # A new unit, a finished one (results written) and a credited one (folder deleted) are saved at once as well.
  urgent_event() { case "$1" in *ISDIR*|*wuresults*|*wudata_*|*checkpt.crc|*/checkpoint|*checkpointState*|*.cpt) return 0 ;; esac; return 1; }
  local ipid
  exec {events}< <(exec inotifywait -m -r -q -e close_write,moved_to,delete --format '%e %w%f' \
    --exclude '(viewerFrame|viewerTop|logfile_)' "$DATA/work" 2>/dev/null)
  ipid=$!
  trap 'kill "$ipid" 2>/dev/null' EXIT
  push; [ $? = 2 ] && exit 2
  last=$(date +%s)
  local checked=$last owner
  while true; do
    urgent=0
    # Saves can be minutes apart, so ownership is checked every minute on its own: a replaced node must stop folding.
    if [ $(( $(date +%s) - checked )) -ge 60 ]; then
      checked=$(date +%s)
      if owner=$(rclone cat "$REMOTE/owner" 2>/dev/null) && [ "$owner" != "$(cat "$OWNER" 2>/dev/null)" ]; then
        log "another container now owns $REMOTE, stopping saves"
        exit 2
      fi
    fi
    read -r -t 60 -u "$events" ev; rc=$?
    if [ "$rc" = 0 ]; then
      urgent_event "$ev" && urgent=1
      until=$(( $(date +%s) + 15 ))
      while [ "$(date +%s)" -lt "$until" ]; do
        read -r -t 10 -u "$events" ev; rc=$?
        [ "$rc" = 0 ] || break
        urgent_event "$ev" && urgent=1
      done
    fi
    # read returns 1 at end of input (inotifywait died) and more than 128 on timeout.
    [ "$rc" = 1 ] && { log "file watcher stopped"; exit 1; }
    now=$(date +%s)
    if [ "$urgent" = 1 ] || [ $((now - last)) -ge 600 ]; then
      push; [ $? = 2 ] && exit 2
      last=$now
    fi
  done
}

"$@"
