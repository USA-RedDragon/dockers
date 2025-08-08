FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

# renovate: datasource=repology depName=alpine_3_22/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r4
# renovate: datasource=repology depName=alpine_3_22/iptables
ARG IPTABLES_VERSION=1.8.11-r1

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
