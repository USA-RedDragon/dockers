FROM alpine:3.23.2@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62 AS builder

# renovate: datasource=repology depName=alpine_3_23/iperf3
ARG IPERF3_VERSION=3.19.1-r0

RUN apk add --no-cache \
    iperf3="${IPERF3_VERSION}"

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
