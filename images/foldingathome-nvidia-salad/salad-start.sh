#!/bin/bash
export PATH="$PATH:/usr/lib/wsl/lib"
echo "salad-start: nvidia-smi=$(command -v nvidia-smi || echo missing) pci_nvidia=$(grep -lsx 0x10de /sys/bus/pci/devices/*/vendor | tr '\n' ' ') resolv=$(grep -m1 nameserver /etc/resolv.conf 2>/dev/null)" >&2
for url in https://api.foldingathome.org/gpus https://s3.mcswain.dev/minio/health/live; do
  echo "salad-start: probe $url -> $(curl -s -o /dev/null -m 15 -w '%{http_code} %{remote_ip} %{time_total}s' "$url" 2>&1)" >&2
done
if { [ -n "$FAH_FAKE_PCI" ] || ! grep -qsx 0x10de /sys/bus/pci/devices/*/vendor; } && command -v nvidia-smi >/dev/null; then
  export FAKE_PCI_ROOT=/tmp/fakepci
  while IFS=', ' read -r bus id; do
    [ -n "$bus" ] || continue
    addr=$(printf '0000:%s' "${bus#*:}" | tr 'A-F' 'a-f')
    mkdir -p "$FAKE_PCI_ROOT/$addr"
    printf '0x%04x\n' $((id & 0xffff)) > "$FAKE_PCI_ROOT/$addr/vendor"
    printf '0x%04x\n' $(((id >> 16) & 0xffff)) > "$FAKE_PCI_ROOT/$addr/device"
    b=${addr#*:}; slotfn=${b#*:}; bus=${b%%:*}; slot=${slotfn%.*}; fn=${slotfn#*.}
    printf '%s%02x\t%04x%04x\t0\n' "$bus" $(( (16#$slot << 3) | fn )) $((id & 0xffff)) $(((id >> 16) & 0xffff)) >> /tmp/fakepci.proc
    echo "salad-start: PCI $addr vendor $(cat "$FAKE_PCI_ROOT/$addr/vendor") device $(cat "$FAKE_PCI_ROOT/$addr/device") from nvidia-smi" >&2
  done < <(nvidia-smi --query-gpu=pci.bus_id,pci.device_id --format=csv,noheader)
  export FAKE_PCI_PROC=/tmp/fakepci.proc LD_PRELOAD=/usr/local/lib/libfakepci.so
fi
salad-sync restore
# Units are bound to a GPU by PCI address, which differs between Salad nodes; move restored units onto this node's GPU.
db=/var/lib/fah-client/client.db
# cbang names GPUs gpu:BUS:SLOT:FUNC in decimal (%02d), while nvidia-smi prints the PCI address in hex.
gpu=""
IFS=':.' read -r _ pbus pslot pfn < <(nvidia-smi --query-gpu=pci.bus_id --format=csv,noheader 2>/dev/null | head -1 | tr -d ' ')
[ -n "$pfn" ] && gpu=$(printf 'gpu:%02d:%02d:%02d' "$((16#$pbus))" "$((16#$pslot))" "$((16#$pfn))")
if [ -f "$db" ] && [ -n "$gpu" ]; then
  for old in $(sqlite3 "$db" "SELECT value FROM units" | grep -oE 'gpu:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}' | sort -u); do
    [ "$old" = "$gpu" ] && continue
    sqlite3 "$db" "UPDATE units SET value = replace(value, '$old', '$gpu')" && echo "salad-start: moved saved units from $old to $gpu" >&2
  done
  sqlite3 "$db" "UPDATE groups SET value = json_set(value, '\$.gpus', json('{\"$gpu\":{\"enabled\":true}}'))"
fi
# The client makes a new identity, orphaning every saved unit, when /etc/machine-id differs from the one it saved; image rebuilds change that file.
mid=$(tr -cd '0-9a-f' < /etc/machine-id)
if [ -f "$db" ] && [ -n "$mid" ]; then
  saved=$(sqlite3 "$db" "SELECT value FROM config WHERE name = 'machine-id'")
  [ -n "$saved" ] && [ "$saved" != "$mid" ] && sqlite3 "$db" "UPDATE config SET value = '$mid' WHERE name = 'machine-id'" \
    && echo "salad-start: kept saved identity across machine-id $saved -> $mid" >&2
fi
args=(--user=USA-RedDragon --team=223518 --cpus=0 --log-to-screen=true)
[ -n "$FAH_PASSKEY" ] && args+=(--passkey="$FAH_PASSKEY")
fah-client "${args[@]}" &
pid=$!
salad-sync watch &
for i in $(seq 60); do (exec 3<>/dev/tcp/127.0.0.1/7396) 2>/dev/null && break; sleep 2; done
salad-ws setup
if [ -n "$FAH_ACCOUNT_TOKEN" ]; then
  for try in 1 2 3 4 5; do
    id=$(cat /tmp/fah-id 2>/dev/null)
    if [ -n "$id" ] && curl -s -m 15 "https://api.foldingathome.org/machine/$id" | grep -q '"mach_name"'; then
      echo "salad-start: account link confirmed for $id" >&2; break
    fi
    echo "salad-start: account link not confirmed for ${id:-unknown id} (try $try), sending it again" >&2
    salad-ws link
  done
fi
wait "$pid"
