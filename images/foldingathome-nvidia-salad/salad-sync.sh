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
export RCLONE_S3_NO_CHECK_BUCKET=true

log() { echo "salad-sync: $*" >&2; }

enabled() { [ -n "${FAH_S3_URL:-}" ] && [ -n "${FAH_S3_ACCESS_KEY:-}" ] && [ -n "${FAH_S3_SECRET_KEY:-}" ]; }

restore() {
  enabled || { log "no S3 settings, starting fresh"; return 0; }
  if rclone lsf "$REMOTE/client.db" >/dev/null 2>&1 && [ -n "$(rclone lsf "$REMOTE/client.db" 2>/dev/null)" ]; then
    rclone copy "$REMOTE" "$DATA" "${EXCLUDE[@]}" --transfers 8 && log "restored $(find "$DATA/work" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l) unit(s) and client.db from $REMOTE"
  else
    log "nothing saved at $REMOTE, starting fresh"
  fi
}

push() {
  mkdir -p "$SNAP" "$DATA/work"
  # fah-client holds client.db with an exclusive lock, so copy it with its WAL and retry if a checkpoint moved it mid-copy.
  rm -f "$SNAP"/client.db*
  for _ in 1 2 3; do
    before=$(stat -c '%Y %s' "$DATA/client.db" 2>/dev/null) || break
    cp "$DATA/client.db" "$SNAP/client.db" && { [ ! -f "$DATA/client.db-wal" ] || cp "$DATA/client.db-wal" "$SNAP/client.db-wal"; }
    [ "$before" = "$(stat -c '%Y %s' "$DATA/client.db")" ] && break
  done
  rclone sync "$DATA/work" "$REMOTE/work" "${EXCLUDE[@]}" --transfers 4 \
    && { [ ! -f "$SNAP/client.db" ] || rclone sync "$SNAP" "$REMOTE" --include 'client.db*'; } \
    && log "saved $(date -u +%H:%M:%S)" || log "save failed"
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
