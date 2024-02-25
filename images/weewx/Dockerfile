FROM debian:12.5-slim@sha256:6bdbd579ba71f6855deecf57e64524921aed6b97ff1e5195436f244d2cb42b12

WORKDIR /app

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

ENV PYTHONUNBUFFERED=1

RUN mkdir -p /var/www/html/weewx /var/lib/weewx /etc/weewx /usr/share/weewx/user

# renovate: datasource=github-releases depName=weewx/weewx
ARG WEEWX_VERSION=v5.0.1

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get -y --no-install-recommends --no-install-suggests install \
        curl \
        gnupg \
        tzdata \
        build-essential \
        ca-certificates \
        libfreetype6-dev \
        git \
        fonts-freefont-ttf \
        python3-pip \
        python3-requests \
        python3-paho-mqtt \
        python3-dateutil \
        python3-ephem \
        python3-dev \
        zlib1g-dev zlib1g \
        nginx \
        s6 \
        make \
        python3-poetry \
        libjpeg62-turbo-dev libjpeg62-turbo \
    && git clone --branch "${WEEWX_VERSION}" https://github.com/weewx/weewx.git /tmp/weewx \
    && cd /tmp/weewx \
    && make pypi-package \
    && cp src/weewx_data/weewx.conf /etc/weewx/weewx.conf \
    && pip install --no-cache-dir --break-system-packages ./dist/*.whl \
    && apt-get -y remove \
        build-essential \
        python3-dev \
        gnupg \
    && apt-get clean && rm -rf /tmp/setup /var/lib/apt/lists/* /tmp/* /var/tmp/*

# renovate: datasource=github-releases depName=chaunceygardiner/weewx-nws
ARG WEEWX_NWS_VERSION=v2.3
# renovate: datasource=github-tags depName=USA-RedDragon/weewx-prometheus
ARG WEEWX_PROMETHEUS_VERSION=v1.1.9
# renovate: datasource=github-releases depName=gjr80/weewx-stackedwindrose
ARG WEEWX_STACKEDWINDROSE_VERSION=v3.0.2
# renovate: sha: datasource=git-refs depName=weewx-mqtt packageName=https://github.com/USA-RedDragon/weewxMQTT branch=master
ARG WEEWX_MQTT_SHA=778c460c96bfa04bc842abdffdca81b58391188d
# renovate: sha: datasource=git-refs depName=weewx-seasons-dark packageName=https://github.com/USA-RedDragon/weewx-seasons-dark branch=main
ARG WEEWX_SEASONS_DARK_SHA=3f2d888d524366f6d977550711a22cfa145d2665

RUN curl -fSsL "https://github.com/chaunceygardiner/weewx-nws/releases/download/${WEEWX_NWS_VERSION}/weewx-nws-${WEEWX_NWS_VERSION/v/}.zip" -o /tmp/weewx-nws.zip && \
    weectl extension install -y /tmp/weewx-nws.zip && \
    rm /tmp/weewx-nws.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewx-prometheus/archive/refs/tags/${WEEWX_PROMETHEUS_VERSION}.zip" -o /tmp/weewx-prometheus.zip && \
    weectl extension install -y /tmp/weewx-prometheus.zip && \
    rm /tmp/weewx-prometheus.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewxMQTT/archive/${WEEWX_MQTT_SHA}.zip" -o /tmp/weewxMQTT.zip && \
    weectl extension install -y /tmp/weewxMQTT.zip && \
    rm /tmp/weewxMQTT.zip && \
    curl -fSsL "https://github.com/gjr80/weewx-stackedwindrose/releases/download/${WEEWX_STACKEDWINDROSE_VERSION}/stackedwindrose-${WEEWX_STACKEDWINDROSE_VERSION/v/}.tar.gz" -o /tmp/stackedwindrose.tar.gz && \
    tar -zxvf /tmp/stackedwindrose.tar.gz -C /tmp && \
    cp /tmp/stackedwindrose/bin/user/stackedwindrose.py /usr/share/weewx/user && \
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
