#!/bin/bash
# Keeps the client's state in S3 so a replacement node can resume the same units.
# Usage: salad-sync restore | salad-sync watch
set -u
DATA=/var/lib/fah-client
SNAP=/tmp/fah-snap
REMOTE="s3:${FAH_S3_BUCKET:-fah-salad}/${FAH_S3_PREFIX:-default}"
EXCLUDE=(--exclude 'viewerFrame*.json' --exclude 'viewerTop.json' --exclude 'logfile_*.txt')

export RCLONE_CONFIG_S3_TYPE=s3 RCLONE_CONFIG_S3_PROVIDER=Minio RCLONE_CONFIG_S3_ENDPOINT="${FAH_S3_URL:-}"
export RCLONE_CONFIG_S3_ACCESS_KEY_ID="${FAH_S3_ACCESS_KEY:-}" RCLONE_CONFIG_S3_SECRET_ACCESS_KEY="${FAH_S3_SECRET_KEY:-}"
export RCLONE_S3_NO_CHECK_BUCKET=true RCLONE_LOG_LEVEL=ERROR

log() { echo "salad-sync: $*" >&2; }

enabled() { [ -n "${FAH_S3_URL:-}" ] && [ -n "${FAH_S3_ACCESS_KEY:-}" ] && [ -n "${FAH_S3_SECRET_KEY:-}" ]; }

SLOT_FILE=/tmp/fah-slot

# Saves alternate between slot-a and slot-b and "current" names the last complete one, so a node killed mid-upload never leaves a mixed checkpoint behind.
restore() {
  enabled || { log "no S3 settings, starting fresh"; return 0; }
  local cur src
  cur=$(rclone cat "$REMOTE/current" 2>/dev/null | tr -cd 'ab')
  if [ -n "$cur" ]; then
    src="$REMOTE/slot-$cur"
  elif [ -n "$(rclone lsf "$REMOTE/client.db" 2>/dev/null)" ]; then
    src="$REMOTE"
  else
    log "nothing saved at $REMOTE, starting fresh"; return 0
  fi
  echo "${cur:-a}" > "$SLOT_FILE"
  rclone copy "$src" "$DATA" "${EXCLUDE[@]}" --exclude '/slot-*/**' --exclude '/current' --transfers 8 \
    && log "restored $(find "$DATA/work" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l) unit(s) and client.db from $src"
}

tree_stamp() { find "$DATA/client.db" "$DATA/client.db-wal" "$DATA/work" -type f ! -name '*.log' ! -name '*.xvg' ! -name 'viewer*' ! -name 'logfile_*' -printf '%p %s %T@\n' 2>/dev/null | sort; }

# Copies the client state to local disk first: the cores append to logs and rewrite checkpoints while rclone uploads, and S3 rejects a file that grows mid-PUT.
snapshot() {
  for _ in 1 2 3 4 5; do
    before=$(tree_stamp)
    rm -rf "$SNAP" && mkdir -p "$SNAP/work"
    cp -a "$DATA/work/." "$SNAP/work/" 2>/dev/null
    for f in client.db client.db-wal; do [ ! -f "$DATA/$f" ] || cp "$DATA/$f" "$SNAP/$f"; done
    [ "$before" = "$(tree_stamp)" ] && return 0
    sleep 2
  done
  return 1
}

push() {
  mkdir -p "$DATA/work"
  snapshot || { log "save skipped, state kept changing"; return 1; }
  local next=a
  [ "$(cat "$SLOT_FILE" 2>/dev/null)" = a ] && next=b
  if rclone sync "$SNAP" "$REMOTE/slot-$next" "${EXCLUDE[@]}" --transfers 4 && echo "$next" | rclone rcat "$REMOTE/current"; then
    echo "$next" > "$SLOT_FILE"
    log "saved $(date -u +%H:%M:%S) to slot-$next"
    if [ -n "$(rclone lsf "$REMOTE/client.db" 2>/dev/null)" ]; then
      rclone purge "$REMOTE/work" 2>/dev/null; rclone delete "$REMOTE" --max-depth 1 --include 'client.db*' 2>/dev/null
    fi
  else
    log "save failed"
  fi
}

watch() {
  enabled || { log "no S3 settings, not saving state"; sleep infinity; }
  while [ ! -d "$DATA/work" ]; do sleep 5; done
  local last=0
  inotifywait -m -r -q -e close_write,moved_to,delete --format '%w%f' \
    --exclude '(viewerFrame|viewerTop|logfile_)' "$DATA/work" 2>/dev/null |
  while true; do
    if read -r -t 600 _; then
      while read -r -t 10 _; do :; done
    fi
    now=$(date +%s)
    if [ $((now - last)) -ge 30 ]; then push; last=$now; fi
  done
}

"$@"
