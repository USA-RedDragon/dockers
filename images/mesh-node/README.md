# Mesh Node

A container image that runs **unmodified [AREDN®](https://www.arednmesh.org/)
firmware** as an RF-less x86-64 node (a "cloud" / tunnel node). It is built by
extracting AREDN®'s official, checksum-verified x86-64 rootfs and running it
unchanged, so the mesh daemons (`arednlink`, `babeld`), the node manager,
`dnsmasq`, and the web UI all come from upstream - no AREDN® code is vendored or
modified here.

This replaces the older hand-built `mesh-base` image (which compiled the mesh
daemons from source and vendored AREDN®'s scripts). For the rolling snapshot
build, see [mesh-node-nightly](../mesh-node-nightly/README.md).

The pinned release is set by `AREDN_VERSION` in the `Dockerfile` (and the image
tag in `.version.sh`); both track AREDN® git tags via renovate.

## Running

The image boots OpenWRT's runtime (`procd` as PID 1), which needs elevated
networking capabilities:

```sh
docker run --rm -it --cap-add NET_ADMIN ghcr.io/usa-reddragon/mesh-node:4.26.1.0
```

The web UI is served on port 80 by `uhttpd`; publish it with `-p 8080:80`.

### Caveats

- **`NET_ADMIN` is required** (not full `--privileged`) for routing, firewall,
  and WireGuard tunnel management.
- WireGuard tunnels need the **host kernel to provide WireGuard**.
- UI operations that assume real hardware are either hidden (radio/RF, WiFi
  scan - AREDN®'s x86-64 profile has no radios) or not meaningful in a container
  (firmware sysupgrade, reboot). Enabling the node's own supernode role is a
  config action, not a UI action.

## Architectures

`linux/amd64` only - AREDN® publishes the x86-64 rootfs as the
container-suitable target.

## Trademark

AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc.
This project is independent and is **not affiliated with, endorsed by, or
sanctioned by** AREDN, Inc.; the name is used only nominatively, to identify the
upstream firmware that this image packages.

This image redistributes the **official, unmodified** AREDN® x86-64 rootfs.
It does not modify AREDN® source or builds, and it is **not an official AREDN®
software release**. AREDN, Inc.'s trademark terms reserve the marks for
officially sanctioned releases (see
[AREDNLicense.txt](https://github.com/aredn/aredn/blob/main/AREDNLicense.txt));
