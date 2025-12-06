FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375

# renovate: datasource=repology depName=alpine_3_23/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r4
# renovate: datasource=repology depName=alpine_3_23/iptables
ARG IPTABLES_VERSION=1.8.11-r1
# renovate: datasource=repology depName=alpine_3_23/tcpdump
ARG TCPDUMP_VERSION=4.99.5-r0

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}" \
    tcpdump="${TCPDUMP_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
