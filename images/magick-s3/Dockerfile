FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

# renovate: datasource=repology depName=alpine_3_23/aws-cli
ARG AWS_CLI_VERSION=2.32.7-r0
# renovate: datasource=repology depName=alpine_3_23/coreutils
ARG COREUTILS_VERSION=9.8-r1
# renovate: datasource=repology depName=alpine_3_23/curl
ARG CURL_VERSION=8.17.0-r1
# renovate: datasource=repology depName=alpine_3_23/imagemagick
ARG IMAGEMAGICK_VERSION=7.1.2.17-r0
# renovate: datasource=repology depName=alpine_3_23/jq
ARG JQ_VERSION=1.8.1-r0
# renovate: datasource=repology depName=alpine_3_23/netpbm
ARG NETPBM_VERSION=11.12.0-r0
# renovate: datasource=repology depName=alpine_3_23/tzdata
ARG TZDATA_VERSION=2026a-r0

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
