FROM alpine:3.24.0@sha256:660e0827bd401543d81323d4886abbd08fda0fe3ba84337837d0b11a67251283

# renovate: datasource=repology depName=alpine_3_24/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20260223-r0
# renovate: datasource=repology depName=alpine_3_24/iptables
ARG IPTABLES_VERSION=1.8.13-r0
# renovate: datasource=repology depName=alpine_3_24/tcpdump
ARG TCPDUMP_VERSION=4.99.6-r1

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}" \
    tcpdump="${TCPDUMP_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
