FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS olsrd-builder

WORKDIR /tmp
RUN mkdir -p /out

ARG OLSRD_VERSION=a1dee87c22d8fc0ffe73ca0d26c1e928ee911eee

RUN apk add --no-cache \
    git \
    gcc \
    linux-headers \
    bison \
    flex \
    make \
    musl-dev

# Build and install olsrd
RUN git clone https://github.com/USA-RedDragon/olsrd.git \
    && cd olsrd \
    && git checkout "${OLSRD_VERSION}" \
    && make prefix=/usr \
    && make prefix=/usr DESTDIR=/out install arprefresh_install txtinfo_install jsoninfo_install dot_draw_install watchdog_install nameservice_install \
    && cd .. \
    && rm -rf olsrd

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS babeld-builder

WORKDIR /tmp
RUN mkdir -p /out

ARG BABELD_VERSION=30ef463bf59ec15d54abab46a3da7583e9e0264c

RUN apk add --no-cache \
    git \
    gcc \
    make \
    musl-dev \
    linux-headers

# Build and install babeld
RUN git clone https://github.com/USA-RedDragon/babeld.git \
    && cd babeld \
    && git checkout "${BABELD_VERSION}" \
    && make \
    && make PREFIX=/usr DESTDIR=/out install.minimal \
    && cd .. \
    && rm -rf babeld

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS ucode-builder

WORKDIR /tmp
RUN mkdir -p /out

ARG UCODE_VERSION=3206dfac81bb896675612b29cea3c8fb822caa7b

RUN apk add --no-cache \
    git \
    gcc \
    make \
    cmake \
    musl-dev \
    zlib-dev \
    json-c-dev \
    linux-headers

RUN git clone https://github.com/jow-/ucode.git \
    && cd ucode \
    && git checkout "${UCODE_VERSION}" \
    && cmake -DUCI_SUPPORT=OFF -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release . \
    && make DESTDIR=/out install \
    && cd .. \
    && rm -rf ucode

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS dnsmasq-builder

WORKDIR /tmp
RUN mkdir -p /out

# We need to build dnsmasq from source due to this bug: https://thekelleys.org.uk/gitweb/?p=dnsmasq.git;a=commit;h=5846f749e5d878b6b5f7c20f6975bc96b95e4aae
# The bug is fixed in dnsmasq 2.92test11, but the latest release in Alpine v3.23 is 2.91.
# renovate: datasource=git-tags depName=https://thekelleys.org.uk/git/dnsmasq.git
ARG DNSMASQ_VERSION=v2.93

RUN apk add --no-cache \
    git \
    gcc \
    make \
    musl-dev \
    linux-headers

# Build and install dnsmasq
RUN git clone https://thekelleys.org.uk/git/dnsmasq.git \
    && cd dnsmasq \
    && git checkout "${DNSMASQ_VERSION}" \
    && make PREFIX=/usr \
    && make PREFIX=/usr DESTDIR=/out install \
    && cd .. \
    && rm -rf dnsmasq

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

COPY --from=olsrd-builder /out /
COPY --from=babeld-builder /out /
COPY --from=ucode-builder /out /
COPY --from=dnsmasq-builder /out /

RUN apk add --no-cache \
    bash \
    curl \
    zlib \
    lzo \
    openssl \
    iptables \
    iproute2 \
    rsyslog \
    logrotate \
    jq \
    gettext \
    wireguard-tools \
    s6 \
    cronie \
    json-c \
    ca-certificates \
    tcpdump \
    inotify-tools

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

COPY --chown=root:root rootfs /
RUN chmod +x /etc/s6/*/run

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
