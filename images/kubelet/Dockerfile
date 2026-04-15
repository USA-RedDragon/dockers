FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS builder

# renovate: datasource=github-tags depName=kubernetes/kubernetes
ARG KUBELET_VERSION=v1.35.4
ARG TARGETARCH
ARG KUBELET_URL=https://dl.k8s.io/release/${KUBELET_VERSION}/bin/linux/${TARGETARCH}/kubelet

RUN wget -q -O /kubelet ${KUBELET_URL} && chmod +x /kubelet

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.7

# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ARG CA_CERTIFICATES_VERSION=20230311+deb12u1
# renovate: datasource=repology versioning=deb depName=debian_12/libcap2
ARG LIBCAP_VERSION=1:2.66-4+deb12u2+b2
# renovate: datasource=repology versioning=deb depName=debian_12/ethtool
ARG ETHTOOL_VERSION=1:6.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/iproute2
ARG IPROUTE_VERSION=6.1.0-3
# renovate: datasource=repology versioning=deb depName=debian_12/nfs-utils
ARG NFS_COMMON_VERSION=1:2.6.2-4+deb12u1
# renovate: datasource=repology versioning=deb depName=debian_12/socat
ARG SOCAT_VERSION=1.7.4.4-2
# renovate: datasource=repology versioning=deb depName=debian_12/util-linux
ARG UTIL_LINUX_VERSION=2.38.1-5+deb12u3

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ca-certificates="${CA_CERTIFICATES_VERSION}" \
        libcap2="${LIBCAP_VERSION}" \
        ethtool="${ETHTOOL_VERSION}" \
        iproute2="${IPROUTE_VERSION}" \
        nfs-common="${NFS_COMMON_VERSION}" \
        socat="${SOCAT_VERSION}" \
        util-linux="${UTIL_LINUX_VERSION}" \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /kubelet /usr/local/bin/kubelet

ENTRYPOINT ["/usr/local/bin/kubelet"]
