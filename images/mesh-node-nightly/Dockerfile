# Mesh node image, nightly snapshot (unmodified AREDN® firmware).
#
# Extracts a pinned AREDN x86-64 snapshot rootfs into a scratch image and runs it
# as-is. AREDN ships a complete OpenWRT-derived rootfs, so the mesh daemons
# (arednlink, babeld), the manager, dnsmasq, and the web UI are all included and
# started by procd. No AREDN code is vendored or rebuilt here - this image is a
# thin wrapper around upstream AREDN firmware.
#
# The snapshot is pinned (and bumped by renovate) so the image content matches
# its published tag. AREDN keeps only the current snapshot on the mirror, so the
# build works while AREDN_SNAPSHOT is the live snapshot; renovate moves the pin
# forward as new snapshots are published.
#
# AREDN publishes the x86-64 rootfs only as an ext4 image, so the builder uses
# debugfs to extract it without mounting (mounting needs privileges that are not
# available during an image build).

# The builder runs on the native build platform; the rootfs is selected explicitly,
# so cross-building needs no emulation.
FROM --platform=$BUILDPLATFORM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS rootfs

ARG TARGETARCH
# renovate: datasource=custom.aredn-snapshot depName=aredn-snapshot
ARG AREDN_SNAPSHOT=20260620-a66a654e

# debugfs (from e2fsprogs-extra) reads the ext4 rootfs image without mounting it.
# renovate: datasource=repology depName=alpine_3_24/e2fsprogs
ARG E2FSPROGS_VERSION=1.47.4-r0
RUN apk add --no-cache e2fsprogs-extra="${E2FSPROGS_VERSION}"

RUN set -eux; \
    [ "${TARGETARCH}" = "amd64" ] || { echo "AREDN image supports linux/amd64 only (got ${TARGETARCH})" >&2; exit 1; }; \
    base="https://downloads.arednmesh.org/snapshots/targets/x86/64"; \
    file="aredn-${AREDN_SNAPSHOT}-x86-64-generic-ext4-rootfs.img.gz"; \
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
