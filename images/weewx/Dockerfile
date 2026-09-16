FROM debian:12.11-slim@sha256:90522eeb7e5923ee2b871c639059537b30521272f10ca86fdbbbb2b75a8c40cd

WORKDIR /app

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

ENV PYTHONUNBUFFERED=1

# renovate: datasource=github-releases depName=weewx/weewx
ARG WEEWX_VERSION=v5.2.0

ARG DEV_PACKAGES=" \
    libfreetype6-dev \
    python3-dev \
    libjpeg62-turbo-dev \
    zlib1g-dev \
"

ARG RUNTIME_DEPS=" \
    ca-certificates \
    curl \
    tzdata \
    libfreetype6 \
    fonts-freefont-ttf \
    python3 \
    libjpeg62-turbo \
    zlib1g \
    python3-requests \
    python3-paho-mqtt \
    python3-dateutil \
    python3-ephem \
    s6 \
    nginx \
    git \
"

ARG BUILD_DEPS=" \
    make \
    python3-pip \
    python3-poetry \
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
ARG WEEWX_NWS_VERSION=v4.5.6
# renovate: datasource=github-tags depName=USA-RedDragon/weewx-prometheus
ARG WEEWX_PROMETHEUS_VERSION=v1.1.9
# renovate: datasource=github-releases depName=gjr80/weewx-stackedwindrose
ARG WEEWX_STACKEDWINDROSE_VERSION=v3.0.2
# renovate: sha: datasource=git-refs depName=weewx-mqtt packageName=https://github.com/USA-RedDragon/weewxMQTT branch=master
ARG WEEWX_MQTT_SHA=778c460c96bfa04bc842abdffdca81b58391188d
# renovate: sha: datasource=git-refs depName=weewx-seasons-dark packageName=https://github.com/USA-RedDragon/weewx-seasons-dark branch=main
ARG WEEWX_SEASONS_DARK_SHA=3f2d888d524366f6d977550711a22cfa145d2665
# renovate: datasource=github-tags depName=teeks99/weewx-json
ARG WEEWX_JSON_VERSION=v1.3

RUN curl -fSsL "https://github.com/chaunceygardiner/weewx-nws/releases/download/${WEEWX_NWS_VERSION}/weewx-nws.zip" -o /tmp/weewx-nws.zip && \
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
