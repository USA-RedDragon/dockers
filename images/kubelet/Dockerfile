FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS builder

# renovate: datasource=github-tags depName=kubernetes/kubernetes
ARG KUBELET_VERSION=v1.37.1
ARG TARGETARCH
ARG KUBELET_URL=https://dl.k8s.io/release/${KUBELET_VERSION}/bin/linux/${TARGETARCH}/kubelet

RUN wget -q -O /kubelet ${KUBELET_URL} && chmod +x /kubelet

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.7@sha256:368abceecc1308e0913a6fd5ab89a513ee4268becefc2a82dbe616462b29a46b

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ca-certificates \
        libcap2 \
        ethtool \
        iproute2 \
        nfs-common \
        socat \
        util-linux \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /kubelet /usr/local/bin/kubelet

ENTRYPOINT ["/usr/local/bin/kubelet"]
