FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

# renovate: datasource=repology depName=alpine_3_21/aws-cli
ARG AWS_CLI_VERSION=2.22.10-r0
# renovate: datasource=repology depName=alpine_3_21/coreutils
ARG COREUTILS_VERSION=9.5-r2
# renovate: datasource=repology depName=alpine_3_21/imagemagick
ARG IMAGEMAGICK_VERSION=7.1.1.41-r0
# renovate: datasource=repology depName=alpine_3_21/netpbm
ARG NETPBM_VERSION=11.8.2-r0
# renovate: datasource=repology depName=alpine_3_21/tzdata
ARG TZDATA_VERSION=2025b-r0

RUN apk add --no-cache \
    aws-cli=${AWS_CLI_VERSION} \
    coreutils=${COREUTILS_VERSION} \
    imagemagick=${IMAGEMAGICK_VERSION} \
    netpbm=${NETPBM_VERSION} \
    tzdata=${TZDATA_VERSION}

USER nobody:nogroup
WORKDIR /home/nobody
