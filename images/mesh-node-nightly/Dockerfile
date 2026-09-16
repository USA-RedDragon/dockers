# Mesh node image, nightly snapshot (unmodified AREDN® firmware).
#
# Extracts a pinned AREDN x86-64 snapshot rootfs into a scratch image and runs it
# as-is. AREDN ships a complete OpenWRT-derived rootfs, so the mesh daemons
# (arednlink, babeld), the manager, dnsmasq, and the web UI are all included and
# started by procd. No AREDN code is vendored or rebuilt here - this image is a
# thin wrapper around upstream AREDN firmware.
#
# AREDN keeps only the current snapshot on the mirror, so the build resolves
# whichever snapshot is live at build time rather than pinning an id that would
# go stale within a day. The image tag comes from .version.sh, which resolves
# the same way; a scheduled rebuild keeps the published image current.
#
# AREDN publishes the x86-64 rootfs only as an ext4 image, so the builder uses
# debugfs to extract it without mounting (mounting needs privileges that are not
# available during an image build).

# The builder runs on the native build platform; the rootfs is selected explicitly,
# so cross-building needs no emulation.
FROM --platform=$BUILDPLATFORM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS rootfs

ARG TARGETARCH
# Empty resolves the live snapshot; set it to build a specific one (only works
# while that snapshot is still on the mirror).
ARG AREDN_SNAPSHOT_VERSION=

# debugfs (from e2fsprogs-extra) reads the ext4 rootfs image without mounting it.
RUN apk add --no-cache e2fsprogs-extra

RUN set -eux; \
    [ "${TARGETARCH}" = "amd64" ] || { echo "AREDN image supports linux/amd64 only (got ${TARGETARCH})" >&2; exit 1; }; \
    base="https://downloads.arednmesh.org/snapshots/targets/x86/64"; \
    snapshot="${AREDN_SNAPSHOT_VERSION:-$(wget -q -O - "${base}/profiles.json" | sed -n 's/.*"version_number": *"\([^"]*\)".*/\1/p')}"; \
    [ -n "${snapshot}" ] || { echo "could not resolve the current AREDN snapshot" >&2; exit 1; }; \
    file="aredn-${snapshot}-x86-64-generic-ext4-rootfs.img.gz"; \
    wget -q -O /tmp/sha256sums "${base}/sha256sums"; \
    wget -q -O /tmp/rootfs.img.gz "${base}/${file}"; \
    echo "$(grep " \*${file}\$" /tmp/sha256sums | cut -d' ' -f1)  /tmp/rootfs.img.gz" | sha256sum -c -; \
    gunzip /tmp/rootfs.img.gz; \
    mkdir -p /rootfs; \
    debugfs -R "rdump / /rootfs" /tmp/rootfs.img; \
    rm -rf /rootfs/lost+found

FROM scratch
COPY --from=rootfs /rootfs/ /

STOPSIGNAL SIGTERM

# Boot AREDN's init (procd). Requires --cap-add NET_ADMIN at runtime; see README.
CMD ["/sbin/init"]
