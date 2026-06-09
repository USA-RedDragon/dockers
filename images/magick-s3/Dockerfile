FROM alpine:3.24.0@sha256:660e0827bd401543d81323d4886abbd08fda0fe3ba84337837d0b11a67251283

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
