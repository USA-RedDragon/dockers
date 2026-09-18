FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

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
