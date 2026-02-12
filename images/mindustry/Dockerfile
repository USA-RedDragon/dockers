FROM amazoncorretto:25.0.2-alpine@sha256:afb37b0939cf8e627e7a18569b661cd3470611e65639d128f7a709d65615482e

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v146

# renovate: datasource=repology depName=alpine_3_22/curl
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
