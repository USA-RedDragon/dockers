FROM alpine:3.24.0@sha256:660e0827bd401543d81323d4886abbd08fda0fe3ba84337837d0b11a67251283 AS builder

# renovate: datasource=repology depName=alpine_3_24/gettext
ARG GETTEXT_VERSION=1.0-r0

RUN apk add --no-cache \
    gettext-envsubst="${GETTEXT_VERSION}"

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
