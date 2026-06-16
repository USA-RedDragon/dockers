FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9

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
