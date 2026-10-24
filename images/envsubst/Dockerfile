FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS builder

RUN apk add --no-cache \
    gettext-envsubst

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
