FROM alpine:3.20.0@sha256:216266c86fc4dcef5619930bd394245824c2af52fd21ba7c6fa0e618657d4c3b

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
