FROM amazoncorretto:17.0.15-alpine@sha256:0dda2abcd0353b8d1ca5089b713c7b8859bd4aacb2677c3d100362007da81eed

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v145.1

# renovate: datasource=repology depName=alpine_3_21/curl
ARG CURL_VERSION=8.12.1-r1

RUN apk add --no-cache --virtual .build-deps \
    curl="${CURL_VERSION}" \
    && curl -fSsL \
        https://github.com/Anuken/Mindustry/releases/download/${MINDUSTRY_VERSION}/server-release.jar \
        -o /mindustry-server.jar \
    && apk del .build-deps

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER nobody:nogroup
WORKDIR /config
WORKDIR /

EXPOSE 6567
EXPOSE 6567/udp

ENTRYPOINT [ "/entrypoint.sh" ]
