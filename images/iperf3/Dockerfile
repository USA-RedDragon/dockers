FROM alpine:3.24.0@sha256:660e0827bd401543d81323d4886abbd08fda0fe3ba84337837d0b11a67251283 AS builder

# renovate: datasource=repology depName=alpine_3_24/iperf3
ARG IPERF3_VERSION=3.20-r0

RUN apk add --no-cache \
    iperf3="${IPERF3_VERSION}"

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
