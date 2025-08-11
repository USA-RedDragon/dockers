FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1 AS builder

# renovate: datasource=repology depName=alpine_3_22/gettext
ARG GETTEXT_VERSION=0.22.5-r0

RUN apk add --no-cache \
    gettext-envsubst="${GETTEXT_VERSION}"

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
