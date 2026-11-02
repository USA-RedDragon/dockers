#!/bin/bash
args=(--user=USA-RedDragon --team=223518 --cpus=0 --log-to-screen=true)
[ -n "$FAH_PASSKEY" ] && args+=(--passkey="$FAH_PASSKEY")
fah-client "${args[@]}" &
pid=$!
for i in $(seq 60); do (exec 3<>/dev/tcp/127.0.0.1/7396) 2>/dev/null && break; sleep 2; done
perl -MIO::Socket::INET -e '
  sub ws { my $s = IO::Socket::INET->new("127.0.0.1:7396") or die "connect: $!";
    print $s "GET /api/websocket HTTP/1.1\r\nHost: 127.0.0.1:7396\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: MDEyMzQ1Njc4OWFiY2RlZg==\r\nSec-WebSocket-Version: 13\r\n\r\n";
    while (<$s>) { last if $_ eq "\r\n" } $s }
  sub send_msg { my ($s, $m) = @_; my $l = length $m; my $k = "abcd";
    my $h = chr(0x81) . ($l < 126 ? chr(0x80 | $l) : chr(0x80 | 126) . pack("n", $l));
    print $s $h . $k . substr($m ^ ($k x (int($l / 4) + 1)), 0, $l) }
  my ($s, %g);
  for my $try (1 .. 40) {
    $s = ws(); my $buf = ""; my $t = time;
    $s->blocking(0); while (time - $t < 2) { my $r; sysread($s, $r, 65536) and $buf .= $r; select(undef, undef, undef, 0.2) }
    while ($buf =~ /"(gpu:[0-9a-fA-F:]+)"\s*:\s*\{/g) { my $id = $1; my $rest = substr($buf, pos($buf), 600); $rest =~ s/"gpu:.*//s; $g{$id} = 1 if $rest =~ /"supported"\s*:\s*true/ }
    last if %g; close $s; sleep 3;
  }
  $s->blocking(1);
  my $gpus = join ",", map { "\"$_\":{\"enabled\":true}" } sort keys %g;
  print STDERR "salad-start: enabling GPUs: ", join(" ", sort keys %g), "\n";
  send_msg($s, "{\"cmd\":\"config\",\"config\":{\"groups\":{\"\":{\"cpus\":0,\"gpus\":{$gpus}}}}}"); sleep 2;
  send_msg($s, "{\"cmd\":\"state\",\"state\":\"" . ($ENV{FAH_STATE} || "fold") . "\"}"); sleep 2;
'
wait "$pid"
