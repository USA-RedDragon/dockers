FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS builder

# renovate: datasource=repology depName=alpine_3_23/iperf3
ARG IPERF3_VERSION=3.19.1-r1

RUN apk add --no-cache \
    iperf3="${IPERF3_VERSION}"

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
