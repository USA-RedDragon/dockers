FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS olsrd-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_24/git
ARG GIT_VERSION=2.54.0-r0
# renovate: datasource=repology depName=alpine_3_24/gcc
ARG GCC_VERSION=15.2.0-r5
# renovate: datasource=repology depName=alpine_3_24/linux-headers
ARG LINUX_HEADERS_VERSION=7.0.0-r1
# renovate: datasource=repology depName=alpine_3_24/bison
ARG BISON_VERSION=3.8.2-r3
# renovate: datasource=repology depName=alpine_3_24/flex
ARG FLEX_VERSION=2.6.4-r8
# renovate: datasource=repology depName=alpine_3_24/make
ARG MAKE_VERSION=4.4.1-r4
# renovate: datasource=repology depName=alpine_3_24/musl
ARG MUSL_VERSION=1.2.6-r2

ARG OLSRD_VERSION=a1dee87c22d8fc0ffe73ca0d26c1e928ee911eee

RUN apk add --no-cache \
    git="${GIT_VERSION}" \
    gcc="${GCC_VERSION}" \
    linux-headers="${LINUX_HEADERS_VERSION}" \
    bison="${BISON_VERSION}" \
    flex="${FLEX_VERSION}" \
    make="${MAKE_VERSION}" \
    musl-dev="${MUSL_VERSION}"

# Build and install olsrd
RUN git clone https://github.com/USA-RedDragon/olsrd.git \
    && cd olsrd \
    && git checkout "${OLSRD_VERSION}" \
    && make prefix=/usr \
    && make prefix=/usr DESTDIR=/out install arprefresh_install txtinfo_install jsoninfo_install dot_draw_install watchdog_install nameservice_install \
    && cd .. \
    && rm -rf olsrd

FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS babeld-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_24/git
ARG GIT_VERSION=2.54.0-r0
# renovate: datasource=repology depName=alpine_3_24/gcc
ARG GCC_VERSION=15.2.0-r5
# renovate: datasource=repology depName=alpine_3_24/linux-headers
ARG LINUX_HEADERS_VERSION=7.0.0-r1
# renovate: datasource=repology depName=alpine_3_24/make
ARG MAKE_VERSION=4.4.1-r4
# renovate: datasource=repology depName=alpine_3_24/musl
ARG MUSL_VERSION=1.2.6-r2

ARG BABELD_VERSION=30ef463bf59ec15d54abab46a3da7583e9e0264c

RUN apk add --no-cache \
    git="${GIT_VERSION}" \
    gcc="${GCC_VERSION}" \
    make="${MAKE_VERSION}" \
    musl-dev="${MUSL_VERSION}" \
    linux-headers="${LINUX_HEADERS_VERSION}"

# Build and install babeld
RUN git clone https://github.com/USA-RedDragon/babeld.git \
    && cd babeld \
    && git checkout "${BABELD_VERSION}" \
    && make \
    && make PREFIX=/usr DESTDIR=/out install.minimal \
    && cd .. \
    && rm -rf babeld

FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS ucode-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_24/git
ARG GIT_VERSION=2.54.0-r0
# renovate: datasource=repology depName=alpine_3_24/gcc
ARG GCC_VERSION=15.2.0-r5
# renovate: datasource=repology depName=alpine_3_24/linux-headers
ARG LINUX_HEADERS_VERSION=7.0.0-r1
# renovate: datasource=repology depName=alpine_3_24/zlib
ARG ZLIB_VERSION=1.3.2-r0
# renovate: datasource=repology depName=alpine_3_24/make
ARG MAKE_VERSION=4.4.1-r4
# renovate: datasource=repology depName=alpine_3_24/musl
ARG MUSL_VERSION=1.2.6-r2
# renovate: datasource=repology depName=alpine_3_24/cmake
ARG CMAKE_VERSION=4.2.3-r0
# renovate: datasource=repology depName=alpine_3_24/json-c
ARG JSONC_VERSION=0.18-r1

ARG UCODE_VERSION=81205a25437a83e3b9e1eb55ac169be84453d53b

RUN apk add --no-cache \
    git="${GIT_VERSION}" \
    gcc="${GCC_VERSION}" \
    make="${MAKE_VERSION}" \
    cmake="${CMAKE_VERSION}" \
    musl-dev="${MUSL_VERSION}" \
    zlib-dev="${ZLIB_VERSION}" \
    json-c-dev="${JSONC_VERSION}" \
    linux-headers="${LINUX_HEADERS_VERSION}"

RUN git clone https://github.com/jow-/ucode.git \
    && cd ucode \
    && git checkout "${UCODE_VERSION}" \
    && cmake -DUCI_SUPPORT=OFF -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release . \
    && make DESTDIR=/out install \
    && cd .. \
    && rm -rf ucode

FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9 AS dnsmasq-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_24/git
ARG GIT_VERSION=2.54.0-r0
# renovate: datasource=repology depName=alpine_3_24/gcc
ARG GCC_VERSION=15.2.0-r5
# renovate: datasource=repology depName=alpine_3_24/linux-headers
ARG LINUX_HEADERS_VERSION=7.0.0-r1
# renovate: datasource=repology depName=alpine_3_24/make
ARG MAKE_VERSION=4.4.1-r4
# renovate: datasource=repology depName=alpine_3_24/musl
ARG MUSL_VERSION=1.2.6-r2

# We need to build dnsmasq from source due to this bug: https://thekelleys.org.uk/gitweb/?p=dnsmasq.git;a=commit;h=5846f749e5d878b6b5f7c20f6975bc96b95e4aae
# The bug is fixed in dnsmasq 2.92test11, but the latest release in Alpine v3.23 is 2.91.
# renovate: datasource=git-tags depName=https://thekelleys.org.uk/git/dnsmasq.git
ARG DNSMASQ_VERSION=v2.93

RUN apk add --no-cache \
    git="${GIT_VERSION}" \
    gcc="${GCC_VERSION}" \
    make="${MAKE_VERSION}" \
    musl-dev="${MUSL_VERSION}" \
    linux-headers="${LINUX_HEADERS_VERSION}"

# Build and install dnsmasq
RUN git clone https://thekelleys.org.uk/git/dnsmasq.git \
    && cd dnsmasq \
    && git checkout "${DNSMASQ_VERSION}" \
    && make PREFIX=/usr \
    && make PREFIX=/usr DESTDIR=/out install \
    && cd .. \
    && rm -rf dnsmasq

FROM alpine:3.24.1@sha256:bec4ccd3817e7c824eb0388971a0b83fab111d586285511ba0266b77e8dc65a9

COPY --from=olsrd-builder /out /
COPY --from=babeld-builder /out /
COPY --from=ucode-builder /out /
COPY --from=dnsmasq-builder /out /

# renovate: datasource=repology depName=alpine_3_24/bash
ARG BASH_VERSION=5.3.9-r1
# renovate: datasource=repology depName=alpine_3_24/curl
ARG CURL_VERSION=8.20.0-r1
# renovate: datasource=repology depName=alpine_3_24/zlib
ARG ZLIB_VERSION=1.3.2-r0
# renovate: datasource=repology depName=alpine_3_24/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_24/openssl
ARG OPENSSL_VERSION=3.5.7-r0
# renovate: datasource=repology depName=alpine_3_24/iptables
ARG IPTABLES_VERSION=1.8.13-r0
# renovate: datasource=repology depName=alpine_3_24/iproute2
ARG IPRROUTE2_VERSION=7.0.0-r0
# renovate: datasource=repology depName=alpine_3_24/rsyslog
ARG RSYSLOG_VERSION=8.2604.0-r0
# renovate: datasource=repology depName=alpine_3_24/logrotate
ARG LOGROTATE_VERSION=3.22.0-r0
# renovate: datasource=repology depName=alpine_3_24/jq
ARG JQ_VERSION=1.8.1-r0
# renovate: datasource=repology depName=alpine_3_24/gettext
ARG GETTEXT_VERSION=1.0-r0
# renovate: datasource=repology depName=alpine_3_24/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20260223-r0
# renovate: datasource=repology depName=alpine_3_24/s6
ARG S6_VERSION=2.15.0.0-r0
# renovate: datasource=repology depName=alpine_3_24/cronie
ARG CRONIE_VERSION=1.7.2-r1
# renovate: datasource=repology depName=alpine_3_24/json-c
ARG JSONC_VERSION=0.18-r1
# renovate: datasource=repology depName=alpine_3_24/ca-certificates
ARG CA_CERTIFICATES_VERSION=20260611-r0
# renovate: datasource=repology depName=alpine_3_24/tcpdump
ARG TCPDUMP_VERSION=4.99.6-r1
# renovate: datasource=repology depName=alpine_3_24/inotify-tools
ARG INOTIFY_TOOLS_VERSION=4.23.9.0-r0

RUN apk add --no-cache \
    bash="${BASH_VERSION}" \
    curl="${CURL_VERSION}" \
    zlib="${ZLIB_VERSION}" \
    lzo="${LZO_VERSION}" \
    openssl="${OPENSSL_VERSION}" \
    iptables="${IPTABLES_VERSION}" \
    iproute2="${IPRROUTE2_VERSION}" \
    rsyslog="${RSYSLOG_VERSION}" \
    logrotate="${LOGROTATE_VERSION}" \
    jq="${JQ_VERSION}" \
    gettext="${GETTEXT_VERSION}" \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    s6="${S6_VERSION}" \
    cronie="${CRONIE_VERSION}" \
    json-c="${JSONC_VERSION}" \
    ca-certificates="${CA_CERTIFICATES_VERSION}" \
    tcpdump="${TCPDUMP_VERSION}" \
    inotify-tools="${INOTIFY_TOOLS_VERSION}"

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

COPY --chown=root:root rootfs /
RUN chmod +x /etc/s6/*/run

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
