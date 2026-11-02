#!/usr/bin/perl
# Configures the local fah-client over its websocket. Usage: salad-ws setup | salad-ws link
# setup waits for a supported GPU, enables it, sets cpus and state, and links the account; link only re-sends the link.
# Writes the client ID to /tmp/fah-id.
use strict;
use warnings;
use IO::Socket::INET;

my $mode = shift // "setup";

# The state frame can arrive in the same read as the handshake reply, so ws() returns those bytes too.
sub ws {
  my $s = IO::Socket::INET->new("127.0.0.1:7396") or die "connect: $!";
  print $s "GET /api/websocket HTTP/1.1\r\nHost: 127.0.0.1:7396\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: MDEyMzQ1Njc4OWFiY2RlZg==\r\nSec-WebSocket-Version: 13\r\n\r\n";
  my $h = "";
  while ($h !~ /\r\n\r\n/) { sysread($s, $h, 4096, length $h) or die "handshake: $!" }
  return ($s, (split /\r\n\r\n/, $h, 2)[1] // "");
}

sub send_msg {
  my ($s, $m) = @_;
  my $l = length $m;
  my $k = "abcd";
  my $h = chr(0x81) . ($l < 126 ? chr(0x80 | $l) : chr(0x80 | 126) . pack("n", $l));
  print $s $h . $k . substr($m ^ ($k x (int($l / 4) + 1)), 0, $l);
}

sub drain {
  my ($s, $secs) = @_;
  my $t = time;
  my $buf = "";
  while (time - $t < $secs) { my $r; sysread($s, $r, 65536) and $buf .= $r; select(undef, undef, undef, 0.2) }
  return $buf;
}

my ($s, %g, $buf);
while (1) {
  unless (eval { ($s, $buf) = ws(); 1 }) {
    print STDERR "salad-ws: $@";
    sleep 3;
    next;
  }
  $s->blocking(0);
  $buf .= drain($s, 2);
  last if $mode eq "link" && $buf =~ /"info"/;
  while ($buf =~ /"(gpu:[0-9a-fA-F:]+)"\s*:\s*\{/g) {
    my $id = $1;
    my $rest = substr($buf, pos($buf), 600);
    $rest =~ s/"gpu:.*//s;
    $g{$id} = 1 if $rest =~ /"supported"\s*:\s*true/;
  }
  last if %g;
  close $s;
  sleep 3;
}

my $i = index($buf, '"info"');
if ($i >= 0 && substr($buf, $i) =~ /"id"\s*:\s*"([A-Za-z0-9_-]{43})"/) {
  my $id = $1;
  if (open(my $f, ">", "/tmp/fah-id")) { print $f "$id\n"; close $f }
}

$s->blocking(1);
if ($mode eq "setup") {
  my $gpus = join ",", map { "\"$_\":{\"enabled\":true}" } sort keys %g;
  print STDERR "salad-start: enabling GPUs: ", join(" ", sort keys %g), "\n";
  send_msg($s, "{\"cmd\":\"config\",\"config\":{\"groups\":{\"\":{\"cpus\":" . ($ENV{FAH_CPUS} || 0) . ",\"gpus\":{$gpus}}}}}");
  send_msg($s, "{\"cmd\":\"state\",\"state\":\"" . ($ENV{FAH_STATE} || "fold") . "\"}");
}
if ($ENV{FAH_ACCOUNT_TOKEN}) {
  my $name = $ENV{FAH_MACHINE_NAME} || "salad";
  send_msg($s, "{\"cmd\":\"link\",\"token\":\"$ENV{FAH_ACCOUNT_TOKEN}\",\"name\":\"$name\"}");
  print STDERR "salad-start: sent account link as $name\n";
}
# The client can be busy for seconds (core downloads and extraction) and drops queued commands if this side closes first.
$s->blocking(0);
drain($s, 90);
close $s;
