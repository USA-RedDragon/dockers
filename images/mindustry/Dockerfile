FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS downloader

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v146

# renovate: datasource=repology depName=alpine_3_24/curl
ARG CURL_VERSION=8.20.0-r1

RUN apk add --no-cache --virtual .build-deps \
    curl="${CURL_VERSION}" \
    && curl -fSsL \
        https://github.com/Anuken/Mindustry/releases/download/${MINDUSTRY_VERSION}/server-release.jar \
        -o /mindustry-server.jar \
    && apk del .build-deps

FROM amazoncorretto:25.0.4-alpine3.22@sha256:7d93179da1c00e18ac3760c90f4ad1b7fc053cf358a6db29028fa5bb022d3043

COPY --from=downloader /mindustry-server.jar /mindustry-server.jar

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER nobody:nogroup
WORKDIR /config
WORKDIR /

EXPOSE 6567
EXPOSE 6567/udp

ENTRYPOINT [ "/entrypoint.sh" ]
