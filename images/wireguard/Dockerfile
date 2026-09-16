FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9

RUN apk add --no-cache \
    wireguard-tools \
    iptables \
    ip6tables \
    tcpdump

VOLUME ["/etc/wireguard"]
EXPOSE 51820/udp

CMD ["wg-quick", "up", "wg0"]
