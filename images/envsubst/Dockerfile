FROM alpine:3.24.2@sha256:31b6477333eb8257db9e5d7c3a7264fd0467928756f0bbcc27d35bea5d28cdbd AS builder

RUN apk add --no-cache \
    gettext-envsubst

COPY template.sh /template.sh

ENTRYPOINT [ "/template.sh" ]
