FROM ubuntu:mantic-20231128@sha256:cbc171ba52575fec0601f01abf6fdec67f8ed227658cacbc10d778ac3b218307

WORKDIR /app

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

ENV PYTHONUNBUFFERED=1

RUN groupadd --system weewx && useradd --system --create-home --gid weewx weewx

RUN mkdir -p /var/www/html/weewx /var/lib/weewx

RUN export DEBIAN_FRONTEND=noninteractive &&\
    apt-get update && \
    apt-get -y --no-install-recommends --no-install-suggests install \
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
        doas \
        libjpeg-dev libjpeg8 && \
    curl -fSsL https://weewx.com/keys.html | gpg --dearmor --output /etc/apt/trusted.gpg.d/weewx.gpg && \
    curl -fSsL https://weewx.com/apt/weewx-python3.list | tee /etc/apt/sources.list.d/weewx.list && \
    apt-get update && \
    apt-get install --no-install-recommends --no-install-suggests -y weewx && \
    apt-get remove -y python3-pillow && \
    python3 -m pip install --no-cache-dir --break-system-packages Pillow==9.0.1 && \
    apt-get -y remove \
        build-essential \
        python3-dev \
        gnupg \
        libjpeg-dev \
        zlib1g-dev && \
    apt-get clean && rm -rf /tmp/setup /var/lib/apt/lists/* /tmp/* /var/tmp/*

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
    wee_extension --install /tmp/weewx-nws.zip && \
    rm /tmp/weewx-nws.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewx-prometheus/archive/refs/tags/${WEEWX_PROMETHEUS_VERSION}.zip" -o /tmp/weewx-prometheus.zip && \
    wee_extension --install /tmp/weewx-prometheus.zip && \
    rm /tmp/weewx-prometheus.zip && \
    curl -fSsL "https://github.com/USA-RedDragon/weewxMQTT/archive/${WEEWX_MQTT_SHA}.zip" -o /tmp/weewxMQTT.zip && \
    wee_extension --install /tmp/weewxMQTT.zip && \
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
RUN usermod -a -G root weewx

RUN chown -R weewx:weewx /var/www/html/weewx /etc/weewx /usr/share/weewx /var/lib/weewx
RUN chmod g+w /var/www/html/weewx /etc/weewx /usr/share/weewx /var/lib/weewx
RUN chmod a+x /etc/s6/*/run

CMD ["/bin/s6-svscan", "/etc/s6"]
