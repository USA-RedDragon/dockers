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
gpu=$(nvidia-smi --query-gpu=pci.bus_id --format=csv,noheader 2>/dev/null | head -1 | tr 'A-F' 'a-f' | awk -F'[:.]' 'NF >= 4 {printf "gpu:%s:%s:%02x", $2, $3, $4}')
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
perl -MIO::Socket::INET -e '
  sub ws { my $s = IO::Socket::INET->new("127.0.0.1:7396") or die "connect: $!";
    print $s "GET /api/websocket HTTP/1.1\r\nHost: 127.0.0.1:7396\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: MDEyMzQ1Njc4OWFiY2RlZg==\r\nSec-WebSocket-Version: 13\r\n\r\n";
    my $h = ""; while ($h !~ /\r\n\r\n/) { sysread($s, $h, 4096, length $h) or die "handshake: $!" }
    ($s, (split /\r\n\r\n/, $h, 2)[1] // "") }
  sub send_msg { my ($s, $m) = @_; my $l = length $m; my $k = "abcd";
    my $h = chr(0x81) . ($l < 126 ? chr(0x80 | $l) : chr(0x80 | 126) . pack("n", $l));
    print $s $h . $k . substr($m ^ ($k x (int($l / 4) + 1)), 0, $l) }
  my ($s, %g);
  # The state frame can arrive in the same read as the handshake reply, so ws() returns those bytes too.
  while (1) {
    ($s, my $buf) = ws(); my $t = time;
    $s->blocking(0); while (time - $t < 2) { my $r; sysread($s, $r, 65536) and $buf .= $r; select(undef, undef, undef, 0.2) }
    while ($buf =~ /"(gpu:[0-9a-fA-F:]+)"\s*:\s*\{/g) { my $id = $1; my $rest = substr($buf, pos($buf), 600); $rest =~ s/"gpu:.*//s; $g{$id} = 1 if $rest =~ /"supported"\s*:\s*true/ }
    last if %g; close $s; sleep 3;
  }
  $s->blocking(1);
  my $gpus = join ",", map { "\"$_\":{\"enabled\":true}" } sort keys %g;
  print STDERR "salad-start: enabling GPUs: ", join(" ", sort keys %g), "\n";
  send_msg($s, "{\"cmd\":\"config\",\"config\":{\"groups\":{\"\":{\"cpus\":" . ($ENV{FAH_CPUS} || 0) . ",\"gpus\":{$gpus}}}}}"); sleep 2;
  send_msg($s, "{\"cmd\":\"state\",\"state\":\"" . ($ENV{FAH_STATE} || "fold") . "\"}"); sleep 2;
  if ($ENV{FAH_ACCOUNT_TOKEN}) {
    my $name = $ENV{FAH_MACHINE_NAME} || "salad";
    send_msg($s, "{\"cmd\":\"link\",\"token\":\"$ENV{FAH_ACCOUNT_TOKEN}\",\"name\":\"$name\"}"); sleep 2;
    print STDERR "salad-start: linked to account as $name\n";
  }
'
wait "$pid"
