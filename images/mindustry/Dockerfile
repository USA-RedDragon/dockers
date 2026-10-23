FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS downloader

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v146

RUN apk add --no-cache --virtual .build-deps \
    curl \
    && curl -fSsL \
        https://github.com/Anuken/Mindustry/releases/download/${MINDUSTRY_VERSION}/server-release.jar \
        -o /mindustry-server.jar \
    && apk del .build-deps

FROM amazoncorretto:25.0.5-alpine3.22@sha256:69e1c9ca5a78941db253ae9093f31f343d1334c124132f5294d33b23e3913b2c

COPY --from=downloader /mindustry-server.jar /mindustry-server.jar

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER nobody:nogroup
WORKDIR /config
WORKDIR /

EXPOSE 6567
EXPOSE 6567/udp

ENTRYPOINT [ "/entrypoint.sh" ]
