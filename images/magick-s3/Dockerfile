FROM alpine:3.24.2@sha256:3cf95fe0816180395592b8373f3ec60663f076127617bbacb4eacf9667afe2e9

RUN apk add --no-cache \
    aws-cli \
    coreutils \
    curl \
    imagemagick \
    jq \
    netpbm \
    tzdata

USER nobody:nogroup
WORKDIR /home/nobody
