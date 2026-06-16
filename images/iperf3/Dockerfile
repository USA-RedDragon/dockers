FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS builder

# renovate: datasource=repology depName=alpine_3_24/iperf3
ARG IPERF3_VERSION=3.20-r0

RUN apk add --no-cache \
    iperf3="${IPERF3_VERSION}"

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
