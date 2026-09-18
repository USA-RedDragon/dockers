FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS downloader

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v159.7

RUN apk add --no-cache --virtual .build-deps \
    curl \
    && curl -fSsL \
        https://github.com/Anuken/Mindustry/releases/download/${MINDUSTRY_VERSION}/server-release.jar \
        -o /mindustry-server.jar \
    && apk del .build-deps

FROM amazoncorretto:25.0.4-alpine3.22@sha256:3c4ad8dd86bd2e0ad068d6e4d7ed13785f923742cd13f9e030988ebf1f41e875

COPY --from=downloader /mindustry-server.jar /mindustry-server.jar

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER nobody:nogroup
WORKDIR /config
WORKDIR /

EXPOSE 6567
EXPOSE 6567/udp

ENTRYPOINT [ "/entrypoint.sh" ]
