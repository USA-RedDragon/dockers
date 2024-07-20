FROM debian:12.6-slim@sha256:39868a6f452462b70cf720a8daff250c63e7342970e749059c105bf7c1e8eeaf

WORKDIR /app

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

ENV PYTHONUNBUFFERED=1

# renovate: datasource=github-releases depName=weewx/weewx
ARG WEEWX_VERSION=v5.0.2

# renovate: datasource=repology versioning=deb depName=debian_12/freetype
ARG FREETYPE_VERSION=2.12.1+dfsg-5+deb12u3
# renovate: datasource=repology versioning=deb depName=debian_12/python3-defaults
ARG PYTHON3_VERSION=3.11.2-1+b1
# renovate: datasource=repology versioning=deb depName=debian_12/libjpeg-turbo
ARG LIBJPEG_TURBO_VERSION=1:2.1.5-2
# renovate: datasource=repology versioning=deb depName=debian_12/zlib
ARG ZLIB_VERSION=1:1.2.13.dfsg-1

ARG DEV_PACKAGES=" \
    libfreetype6-dev="${FREETYPE_VERSION}" \
    python3-dev="${PYTHON3_VERSION}" \
    libjpeg62-turbo-dev="${LIBJPEG_TURBO_VERSION}" \
    zlib1g-dev="${ZLIB_VERSION}" \
"

# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ARG CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/curl
ARG CURL_VERSION=7.88.1-10+deb12u6
# renovate: datasource=repology versioning=deb depName=debian_12/tzdata
ARG TZDATA_VERSION=2024a-0+deb12u1
# renovate: datasource=repology versioning=deb depName=debian_12/fonts-freefont
ARG FONTS_FREEFONT_VERSION=20120503-10
# renovate: datasource=repology versioning=deb depName=debian_12/s6
ARG S6_VERSION=2.11.3.2-1
# renovate: datasource=repology versioning=deb depName=debian_12/nginx
ARG NGINX_VERSION=1.22.1-9
# renovate: datasource=repology versioning=deb depName=debian_12/requests
ARG REQUESTS_VERSION=2.28.1+dfsg-1
# renovate: datasource=repology versioning=deb depName=debian_12/python-paho-mqtt
ARG PAHO_MQTT_VERSION=1.6.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/python-dateutil
ARG DATEUTIL_VERSION=2.8.2-2
# renovate: datasource=repology versioning=deb depName=debian_12/pyephem
ARG PYEPHEM_VERSION=4.1.4-2+b1
# renovate: datasource=repology versioning=deb depName=debian_12/git
ARG GIT_VERSION=1:2.39.2-1.1

ARG RUNTIME_DEPS=" \
    ca-certificates="${CA_CERTIFICATES_VERSION}" \
    curl="${CURL_VERSION}" \
    tzdata="${TZDATA_VERSION}" \
    libfreetype6="${FREETYPE_VERSION}" \
    fonts-freefont-ttf="${FONTS_FREEFONT_VERSION}" \
    python3="${PYTHON3_VERSION}" \
    libjpeg62-turbo="${LIBJPEG_TURBO_VERSION}" \
    zlib1g="${ZLIB_VERSION}" \
    python3-requests="${REQUESTS_VERSION}" \
    python3-paho-mqtt="${PAHO_MQTT_VERSION}" \
    python3-dateutil="${DATEUTIL_VERSION}" \
    python3-ephem="${PYEPHEM_VERSION}" \
    s6="${S6_VERSION}" \
    nginx="${NGINX_VERSION}" \
    git=${GIT_VERSION} \
"

# renovate: datasource=repology versioning=deb depName=debian_12/make-dfsg
ARG MAKE_VERSION=4.3-4.1
# renovate: datasource=repology versioning=deb depName=debian_12/python-pip
ARG PIP_VERSION=23.0.1+dfsg-1
# renovate: datasource=repology versioning=deb depName=debian_12/poetry
ARG POETRY_VERSION=1.3.2+dfsg-3

ARG BUILD_DEPS=" \
    make=${MAKE_VERSION} \
    python3-pip="${PIP_VERSION}" \
    python3-poetry="${POETRY_VERSION}" \
"

RUN mkdir -p /var/www/html/weewx /var/lib/weewx /etc/weewx /usr/share/weewx/user

# False-positive from hadolint
# hadolint ignore=DL3008
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get -y --no-install-recommends --no-install-suggests install \
        ${DEV_PACKAGES} \
        ${BUILD_DEPS} \
        ${RUNTIME_DEPS} \
    && git clone --branch "${WEEWX_VERSION}" https://github.com/weewx/weewx.git /tmp/weewx \
    && cd /tmp/weewx \
    && make pypi-package \
    && cp src/weewx_data/weewx.conf /etc/weewx/weewx.conf \
    && pip install --no-cache-dir --break-system-packages ./dist/*.whl \
    && apt-get -y remove --purge \
        ${DEV_PACKAGES} \
        ${BUILD_DEPS} \
    && apt-get clean && rm -rf /tmp/setup /var/lib/apt/lists/* /tmp/* /var/tmp/*

# renovate: datasource=github-releases depName=chaunceygardiner/weewx-nws
ARG WEEWX_NWS_VERSION=v4.5
# renovate: datasource=github-tags depName=USA-RedDragon/weewx-prometheus
ARG WEEWX_PROMETHEUS_VERSION=v1.1.9
# renovate: datasource=github-releases depName=gjr80/weewx-stackedwindrose
ARG WEEWX_STACKEDWINDROSE_VERSION=v3.0.2
# renovate: sha: datasource=git-refs depName=weewx-mqtt packageName=https://github.com/USA-RedDragon/weewxMQTT branch=master
ARG WEEWX_MQTT_SHA=778c460c96bfa04bc842abdffdca81b58391188d
# renovate: sha: datasource=git-refs depName=weewx-seasons-dark packageName=https://github.com/USA-RedDragon/weewx-seasons-dark branch=main
ARG WEEWX_SEASONS_DARK_SHA=3f2d888d524366f6d977550711a22cfa145d2665
# renovate: datasource=github-tags depName=teeks99/weewx-json
ARG WEEWX_JSON_VERSION=v1.2

RUN curl -fSsL "https://github.com/chaunceygardiner/weewx-nws/releases/download/${WEEWX_NWS_VERSION}/weewx-nws-${WEEWX_NWS_VERSION/v/}.zip" -o /tmp/weewx-nws.zip && \
    weectl extension install -y /tmp/weewx-nws.zip && \
    rm /tmp/weewx-nws.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewx-prometheus/archive/refs/tags/${WEEWX_PROMETHEUS_VERSION}.zip" -o /tmp/weewx-prometheus.zip && \
    weectl extension install -y /tmp/weewx-prometheus.zip && \
    rm /tmp/weewx-prometheus.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewxMQTT/archive/${WEEWX_MQTT_SHA}.zip" -o /tmp/weewxMQTT.zip && \
    weectl extension install -y /tmp/weewxMQTT.zip && \
    rm /tmp/weewxMQTT.zip && \
    curl -fSsL "https://github.com/teeks99/weewx-json/releases/download/${WEEWX_JSON_VERSION}/weewx-json_${WEEWX_JSON_VERSION/v/}.tar.gz" -o /tmp/weewx-json.tar.gz && \
    weectl extension install -y /tmp/weewx-json.tar.gz && \
    rm /tmp/weewx-json.tar.gz && \
    curl -fSsL "https://github.com/gjr80/weewx-stackedwindrose/releases/download/${WEEWX_STACKEDWINDROSE_VERSION}/stackedwindrose-${WEEWX_STACKEDWINDROSE_VERSION/v/}.tar.gz" -o /tmp/stackedwindrose.tar.gz && \
    tar -zxvf /tmp/stackedwindrose.tar.gz -C /tmp && \
    cp /tmp/stackedwindrose/bin/user/stackedwindrose.py /etc/weewx/bin/user && \
    cp -R /tmp/stackedwindrose/skins/* /etc/weewx/skins && \
    rm -rf /tmp/stackedwindrose.tar.gz /tmp && \
    rm -rf /etc/weewx/skins/Seasons/ && \
    git clone https://github.com/USA-RedDragon/weewx-seasons-dark.git && \
    cd weewx-seasons-dark && \
    git checkout "${WEEWX_SEASONS_DARK_SHA}" && \
    cd .. && \
    mv weewx-seasons-dark/skins/Seasons /etc/weewx/skins/ && \
    rm -rf weewx-seasons-dark

COPY --chown=root:root rootfs /

RUN groupadd weewx && useradd --create-home --gid weewx weewx
RUN usermod -a -G root weewx

RUN chown -R weewx:weewx /var/www/html/weewx /etc/weewx /usr/share/weewx /var/lib/weewx
RUN chmod g+w /var/www/html/weewx /etc/weewx /usr/share/weewx /var/lib/weewx
RUN chmod +x /etc/s6/*/run

CMD ["/bin/s6-svscan", "/etc/s6"]
