#!/bin/bash
export PATH="$PATH:/usr/lib/wsl/lib"
log() { echo "salad-start: $*" >&2; }
log "nvidia-smi=$(command -v nvidia-smi || echo missing) pci_nvidia=$(grep -lsx 0x10de /sys/bus/pci/devices/*/vendor | tr '\n' ' ') resolv=$(grep -m1 nameserver /etc/resolv.conf 2>/dev/null)"
for url in https://api.foldingathome.org/gpus https://s3.mcswain.dev/minio/health/live; do
  log "probe $url -> $(curl -s -o /dev/null -m 15 -w '%{http_code} %{remote_ip} %{time_total}s' "$url" 2>&1)"
done
# Starting the client without the saved state would give it a new identity, and the next save would overwrite the old one.
salad-sync restore || { log "saved state not restored, exiting so Salad retries"; exit 1; }

db=/var/lib/fah-client/client.db
# A unit's GPU name (gpu:BUS:SLOT:FUNC, decimal) is part of its signed assignment, and editing it gets the result
# rejected, so units are never changed. Instead the client is shown the GPU at the address the oldest waiting or
# running GPU unit was assigned with, or at 01:00.0 for new work. Units assigned elsewhere wait for a later start.
gpu=gpu:01:00:00
if [ -f "$db" ]; then
  want=$(sqlite3 "$db" "SELECT json_extract(value, '\$.data.assignment.data.gpus[0]') FROM units
    WHERE json_extract(value, '\$.state.state') IN ('CORE', 'RUN') AND json_extract(value, '\$.data.assignment.data.gpus[0]') IS NOT NULL
    ORDER BY json_extract(value, '\$.data.assignment.data.time') LIMIT 1")
  [[ "$want" =~ ^gpu:[0-9]{2,3}:[0-9]{2}:00$ ]] && gpu=$want
fi
IFS=: read -r _ gbus gslot _ <<<"$gpu"
gbus=$((10#$gbus)); gslot=$((10#$gslot))
if { [ -n "$FAH_FAKE_PCI" ] || ! grep -qsx 0x10de /sys/bus/pci/devices/*/vendor; } && command -v nvidia-smi >/dev/null; then
  export FAKE_PCI_ROOT=/tmp/fakepci FAKE_GPU_BUS=$gbus FAKE_GPU_SLOT=$gslot
  IFS=', ' read -r rbus id < <(nvidia-smi --query-gpu=pci.bus_id,pci.device_id --format=csv,noheader | head -1)
  addr=$(printf '0000:%02x:%02x.0' "$gbus" "$gslot")
  mkdir -p "$FAKE_PCI_ROOT/$addr"
  printf '0x%04x\n' $((id & 0xffff)) > "$FAKE_PCI_ROOT/$addr/vendor"
  printf '0x%04x\n' $(((id >> 16) & 0xffff)) > "$FAKE_PCI_ROOT/$addr/device"
  printf '%02x%02x\t%04x%04x\t0\n' "$gbus" $((gslot << 3)) $((id & 0xffff)) $(((id >> 16) & 0xffff)) > /tmp/fakepci.proc
  log "GPU at $rbus presented as $gpu (PCI $addr, device $(cat "$FAKE_PCI_ROOT/$addr/device"))"
  export FAKE_PCI_PROC=/tmp/fakepci.proc LD_PRELOAD=/usr/local/lib/libfakepci.so
fi
if [ -f "$db" ]; then
  sqlite3 "$db" "UPDATE groups SET value = json_set(value, '\$.gpus', json('{\"$gpu\":{\"enabled\":true}}'))"
  # The client makes a new identity, orphaning every saved unit, when /etc/machine-id differs from the one it saved; image rebuilds change that file.
  mid=$(tr -cd '0-9a-f' < /etc/machine-id)
  saved=$(sqlite3 "$db" "SELECT value FROM config WHERE name = 'machine-id'")
  if [ -n "$mid" ] && [ -n "$saved" ] && [ "$saved" != "$mid" ]; then
    sqlite3 "$db" "UPDATE config SET value = '$mid' WHERE name = 'machine-id'" && log "kept saved identity across machine-id $saved -> $mid"
  fi
  # fah-client pauses a group by itself after more than 4 units fail (a broken GPU) and saves that pause, so every later
  # node would sit idle. A new node clears it; a pause without such a failure streak is the user's and is kept.
  failed=$(sqlite3 "$db" "SELECT count(*) FROM (SELECT json_extract(value, '\$.result') r FROM wu_log ORDER BY ts DESC LIMIT 5) WHERE r != 'credited'")
  paused=$(sqlite3 "$db" "SELECT json_extract(value, '\$.paused') FROM groups WHERE name = ''")
  if [ "$failed" = 5 ] && [ "$paused" = 1 ]; then
    sqlite3 "$db" "UPDATE groups SET value = json_set(value, '\$.paused', json('false')) WHERE name = ''" &&
      log "cleared the pause the client set after 5 failed units on a previous node"
  fi
  # Change times saved on a node whose clock ran ahead would make this client ignore config and state commands.
  sqlite3 "$db" "DELETE FROM config WHERE name LIKE 'change-time-%'"
fi

args=(--user=USA-RedDragon --team=223518 --cpus=0 --log-to-screen=true)
[ -n "$FAH_PASSKEY" ] && args+=(--passkey="$FAH_PASSKEY")
fah-client "${args[@]}" &
pid=$!
salad-sync watch &
wpid=$!

configure() {
  for i in $(seq 60); do (exec 3<>/dev/tcp/127.0.0.1/7396) 2>/dev/null && break; sleep 2; done
  if ! timeout 1800 salad-ws setup; then
    log "client setup failed, stopping the client"
    kill -TERM "$pid"
    return 1
  fi
  [ -n "$FAH_ACCOUNT_TOKEN" ] || return 0
  for try in 1 2 3 4 5; do
    id=$(cat /tmp/fah-id 2>/dev/null)
    if [ -n "$id" ] && curl -s -m 15 "https://api.foldingathome.org/machine/$id" | grep -q '"mach_name"'; then
      log "account link confirmed for $id"; return 0
    fi
    log "account link not confirmed for ${id:-unknown id} (try $try), sending it again"
    timeout 300 salad-ws link
  done
}
# A node that starts while the fleet is winding down finishes what it restored instead of fetching new work.
if [ "$(salad-sync control)" = finish ]; then
  export FAH_STATE=finish
  touch /tmp/fah-finish-sent
  log "fleet is finishing, completing restored units only"
fi
configure &
cpid=$!

# Salad sends SIGTERM before it takes the node away: let the client stop its cores cleanly, then save once more.
stop() {
  log "stopping: $1"
  kill "$cpid" 2>/dev/null
  kill -TERM "$pid" 2>/dev/null
  wait "$pid" 2>/dev/null
  if [ "$2" = save ]; then
    kill "$wpid" 2>/dev/null
    timeout 120 salad-sync push
  fi
  kill "$wpid" 2>/dev/null
  exit "$3"
}
trap 'stop "SIGTERM received" save 0' TERM INT

wait -n "$pid" "$wpid"; wrc=$?
if ! kill -0 "$pid" 2>/dev/null; then
  log "fah-client exited"
  kill "$cpid" "$wpid" 2>/dev/null
  timeout 120 salad-sync push
  exit 1
fi
if [ "$wrc" = 2 ]; then
  stop "another container took over this state" nosave 0
fi
stop "state watcher exited ($wrc)" save 1
