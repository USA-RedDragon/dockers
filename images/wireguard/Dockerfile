FROM alpine:3.19.1@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0

# renovate: datasource=repology depName=alpine_3_19/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r3
# renovate: datasource=repology depName=alpine_3_19/iptables
ARG IPTABLES_VERSION=1.8.10-r3

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
