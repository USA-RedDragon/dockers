FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS builder

# renovate: datasource=repology depName=alpine_3_23/iperf3
ARG IPERF3_VERSION=3.19.1-r0

RUN apk add --no-cache \
    iperf3="${IPERF3_VERSION}"

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
