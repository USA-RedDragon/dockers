FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

# renovate: datasource=repology depName=alpine_3_23/aws-cli
ARG AWS_CLI_VERSION=2.22.10-r0
# renovate: datasource=repology depName=alpine_3_23/coreutils
ARG COREUTILS_VERSION=9.5-r2
# renovate: datasource=repology depName=alpine_3_23/curl
ARG CURL_VERSION=8.12.1-r1
# renovate: datasource=repology depName=alpine_3_23/imagemagick
ARG IMAGEMAGICK_VERSION=7.1.1.41-r0
# renovate: datasource=repology depName=alpine_3_23/jq
ARG JQ_VERSION=1.7.1-r0
# renovate: datasource=repology depName=alpine_3_23/netpbm
ARG NETPBM_VERSION=11.8.2-r0
# renovate: datasource=repology depName=alpine_3_23/tzdata
ARG TZDATA_VERSION=2025b-r0

RUN apk add --no-cache \
    aws-cli=${AWS_CLI_VERSION} \
    coreutils=${COREUTILS_VERSION} \
    curl=${CURL_VERSION} \
    imagemagick=${IMAGEMAGICK_VERSION} \
    jq=${JQ_VERSION} \
    netpbm=${NETPBM_VERSION} \
    tzdata=${TZDATA_VERSION}

USER nobody:nogroup
WORKDIR /home/nobody
