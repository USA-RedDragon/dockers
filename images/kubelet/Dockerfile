FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS builder

# renovate: datasource=github-tags depName=kubernetes/kubernetes
ARG KUBELET_VERSION=v1.37.0
ARG TARGETARCH
ARG KUBELET_URL=https://dl.k8s.io/release/${KUBELET_VERSION}/bin/linux/${TARGETARCH}/kubelet

RUN wget -q -O /kubelet ${KUBELET_URL} && chmod +x /kubelet

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.7

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
