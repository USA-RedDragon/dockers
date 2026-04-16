FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

# renovate: datasource=repology depName=alpine_3_23/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20250521-r1
# renovate: datasource=repology depName=alpine_3_23/iptables
ARG IPTABLES_VERSION=1.8.11-r1
# renovate: datasource=repology depName=alpine_3_23/tcpdump
ARG TCPDUMP_VERSION=4.99.5-r1

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}" \
    tcpdump="${TCPDUMP_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
