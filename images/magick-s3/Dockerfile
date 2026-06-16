FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9

# renovate: datasource=repology depName=alpine_3_24/aws-cli
ARG AWS_CLI_VERSION=2.34.63-r0
# renovate: datasource=repology depName=alpine_3_24/coreutils
ARG COREUTILS_VERSION=9.11-r0
# renovate: datasource=repology depName=alpine_3_24/curl
ARG CURL_VERSION=8.20.0-r1
# renovate: datasource=repology depName=alpine_3_24/imagemagick
ARG IMAGEMAGICK_VERSION=7.1.2.24-r0
# renovate: datasource=repology depName=alpine_3_24/jq
ARG JQ_VERSION=1.8.1-r0
# renovate: datasource=repology depName=alpine_3_24/netpbm
ARG NETPBM_VERSION=11.14.0-r0
# renovate: datasource=repology depName=alpine_3_24/tzdata
ARG TZDATA_VERSION=2026b-r0

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
