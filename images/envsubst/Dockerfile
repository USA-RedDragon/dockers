FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS builder

# renovate: datasource=repology depName=alpine_3_23/gettext
ARG GETTEXT_VERSION=0.22.5-r0

RUN apk add --no-cache \
    gettext-envsubst="${GETTEXT_VERSION}"

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
