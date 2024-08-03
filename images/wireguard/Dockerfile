FROM alpine:3.20.2@sha256:eddacbc7e24bf8799a4ed3cdcfa50d4b88a323695ad80f317b6629883b2c2a78

# renovate: datasource=repology depName=alpine_3_20/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r4
# renovate: datasource=repology depName=alpine_3_20/iptables
ARG IPTABLES_VERSION=1.8.10-r3

RUN apk add --no-cache \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    ip6tables="${IPTABLES_VERSION}"

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
