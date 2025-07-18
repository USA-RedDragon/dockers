FROM amazoncorretto:21.0.7-alpine@sha256:dc8347edc98774eb02b7adc84c65bd46e1fdbebd0acb070d27f7f033341dea99

# renovate: datasource=github-releases depName=Anuken/Mindustry
ARG MINDUSTRY_VERSION=v146

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
