FROM alpine:3.24.2@sha256:3cf95fe0816180395592b8373f3ec60663f076127617bbacb4eacf9667afe2e9 AS builder

RUN apk add --no-cache \
    iperf3

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
