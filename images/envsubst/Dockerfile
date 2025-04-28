FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS builder

# renovate: datasource=repology depName=alpine_3_21/gettext
ARG GETTEXT_VERSION=0.22.5-r0

RUN apk add --no-cache \
    gettext-envsubst="${GETTEXT_VERSION}"

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
