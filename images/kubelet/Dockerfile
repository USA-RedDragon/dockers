FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 AS builder

# renovate: datasource=github-tags depName=kubernetes/kubernetes
ARG KUBELET_VERSION=v1.31.0
ARG TARGETARCH
ARG KUBELET_URL=https://dl.k8s.io/release/${KUBELET_VERSION}/bin/linux/${TARGETARCH}/kubelet

RUN wget -q -O /kubelet ${KUBELET_URL} && chmod +x /kubelet

FROM registry.k8s.io/build-image/debian-iptables:bookworm-v1.0.0@sha256:5f57958e254319ca94004b9deab14c62acc265969d741cd2e5a702ddf0c9e540

# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ARG CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/libcap2
ARG LIBCAP_VERSION=1:2.66-4
# renovate: datasource=repology versioning=deb depName=debian_12/ethtool
ARG ETHTOOL_VERSION=1:6.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/iproute2
ARG IPROUTE_VERSION=6.1.0-3
# renovate: datasource=repology versioning=deb depName=debian_12/nfs-utils
ARG NFS_COMMON_VERSION=1:2.6.2-4
# renovate: datasource=repology versioning=deb depName=debian_12/socat
ARG SOCAT_VERSION=1.7.4.4-2
# renovate: datasource=repology versioning=deb depName=debian_12/util-linux
ARG UTIL_LINUX_VERSION=2.38.1-5+deb12u1
# renovate: datasource=repology versioning=deb depName=debian_12/bash
ARG BASH_VERSION=5.2.15-2+b7
# renovate: datasource=repology versioning=deb depName=debian_12/ceph
ARG CEPH_VERSION=16.2.11+ds-2
# renovate: datasource=repology versioning=deb depName=debian_12/cifs-utils
ARG CIFS_UTILS_VERSION=2:7.0-2
# renovate: datasource=repology versioning=deb depName=debian_12/e2fsprogs
ARG E2FSPROGS_VERSION=1.47.0-2
# renovate: datasource=repology versioning=deb depName=debian_12/glusterfs
ARG GLUSTERFS_VERSION=10.3-5
# renovate: datasource=repology versioning=deb depName=debian_12/jq
ARG JQ_VERSION=1.6-2.1
# renovate: datasource=repology versioning=deb depName=debian_12/procps
ARG PROCPS_VERSION=2:4.0.2-3
# renovate: datasource=repology versioning=deb depName=debian_12/ucf
ARG UCF_VERSION=3.0043+nmu1
# renovate: datasource=repology versioning=deb depName=debian_12/udev
ARG UDEV_VERSION=252.26-1~deb12u2
# renovate: datasource=repology versioning=deb depName=debian_12/xfsprogs
ARG XFSPROGS_VERSION=6.1.0-1

RUN clean-install \
  --allow-change-held-packages \
  ca-certificates="${CA_CERTIFICATES_VERSION}" \
  libcap2="${LIBCAP_VERSION}" \
  ethtool="${ETHTOOL_VERSION}" \
  iproute2="${IPROUTE_VERSION}" \
  nfs-common="${NFS_COMMON_VERSION}" \
  socat="${SOCAT_VERSION}" \
  util-linux="${UTIL_LINUX_VERSION}" \
  bash="${BASH_VERSION}" \
  ceph-common="${CEPH_VERSION}" \
  cifs-utils="${CIFS_UTILS_VERSION}" \
  e2fsprogs="${E2FSPROGS_VERSION}" \
  glusterfs-client="${GLUSTERFS_VERSION}" \
  jq="${JQ_VERSION}" \
  procps="${PROCPS_VERSION}" \
  ucf="${UCF_VERSION}" \
  udev="${UDEV_VERSION}" \
  xfsprogs="${XFSPROGS_VERSION}"

COPY --from=builder /kubelet /usr/local/bin/kubelet

# Add wrapper for iscsiadm
RUN <<__DOCKER_EOF__
set -e
cat > /usr/local/sbin/iscsiadm <<__EOF__
#!/usr/bin/sh

iscsid_pid=\$(pgrep iscsid)

if [ -z "\${iscsid_pid}"]; then
    >&2 echo ERROR: Cannot find iscsid PID
    exit 1
fi

exec nsenter --target=\${iscsid_pid} --mount --net -- /usr/local/sbin/iscsiadm "\$@"
__EOF__
chmod +x /usr/local/sbin/iscsiadm
__DOCKER_EOF__

ENTRYPOINT ["/usr/local/bin/kubelet"]
