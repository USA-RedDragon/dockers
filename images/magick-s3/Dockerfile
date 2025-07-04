FROM alpine:3.22.0@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715

# renovate: datasource=repology depName=alpine_3_22/aws-cli
ARG AWS_CLI_VERSION=2.22.10-r0
# renovate: datasource=repology depName=alpine_3_22/coreutils
ARG COREUTILS_VERSION=9.5-r2
# renovate: datasource=repology depName=alpine_3_22/curl
ARG CURL_VERSION=8.12.1-r1
# renovate: datasource=repology depName=alpine_3_22/imagemagick
ARG IMAGEMAGICK_VERSION=7.1.1.41-r0
# renovate: datasource=repology depName=alpine_3_22/jq
ARG JQ_VERSION=1.7.1-r0
# renovate: datasource=repology depName=alpine_3_22/netpbm
ARG NETPBM_VERSION=11.8.2-r0
# renovate: datasource=repology depName=alpine_3_22/tzdata
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
