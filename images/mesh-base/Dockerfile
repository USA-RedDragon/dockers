FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS olsrd-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_22/git
ARG GIT_VERSION=2.52.0-r0
# renovate: datasource=repology depName=alpine_3_22/gcc
ARG GCC_VERSION=15.2.0-r2
# renovate: datasource=repology depName=alpine_3_22/linux-headers
ARG LINUX_HEADERS_VERSION=6.16.12-r0
# renovate: datasource=repology depName=alpine_3_22/bison
ARG BISON_VERSION=3.8.2-r2
# renovate: datasource=repology depName=alpine_3_22/flex
ARG FLEX_VERSION=2.6.4-r7
# renovate: datasource=repology depName=alpine_3_22/make
ARG MAKE_VERSION=4.4.1-r3
# renovate: datasource=repology depName=alpine_3_22/musl
ARG MUSL_VERSION=1.2.5-r21

ARG OLSRD_VERSION=73dc4a6cbf618c961b5723ecd6055102e6855985

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

FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS babeld-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_22/git
ARG GIT_VERSION=2.52.0-r0
# renovate: datasource=repology depName=alpine_3_22/gcc
ARG GCC_VERSION=15.2.0-r2
# renovate: datasource=repology depName=alpine_3_22/linux-headers
ARG LINUX_HEADERS_VERSION=6.16.12-r0
# renovate: datasource=repology depName=alpine_3_22/make
ARG MAKE_VERSION=4.4.1-r3
# renovate: datasource=repology depName=alpine_3_22/musl
ARG MUSL_VERSION=1.2.5-r21

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

FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS ucode-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_22/git
ARG GIT_VERSION=2.52.0-r0
# renovate: datasource=repology depName=alpine_3_22/gcc
ARG GCC_VERSION=15.2.0-r2
# renovate: datasource=repology depName=alpine_3_22/linux-headers
ARG LINUX_HEADERS_VERSION=6.16.12-r0
# renovate: datasource=repology depName=alpine_3_22/zlib
ARG ZLIB_VERSION=1.3.1-r2
# renovate: datasource=repology depName=alpine_3_22/make
ARG MAKE_VERSION=4.4.1-r3
# renovate: datasource=repology depName=alpine_3_22/musl
ARG MUSL_VERSION=1.2.5-r21
# renovate: datasource=repology depName=alpine_3_22/cmake
ARG CMAKE_VERSION=4.1.3-r0
# renovate: datasource=repology depName=alpine_3_22/json-c
ARG JSONC_VERSION=0.18-r1

ARG UCODE_VERSION=f7c2b97a82e8b505bf4b2c0d8883b5116e1960f9

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

FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375 AS dnsmasq-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_22/git
ARG GIT_VERSION=2.52.0-r0
# renovate: datasource=repology depName=alpine_3_22/gcc
ARG GCC_VERSION=15.2.0-r2
# renovate: datasource=repology depName=alpine_3_22/linux-headers
ARG LINUX_HEADERS_VERSION=6.16.12-r0
# renovate: datasource=repology depName=alpine_3_22/make
ARG MAKE_VERSION=4.4.1-r3
# renovate: datasource=repology depName=alpine_3_22/musl
ARG MUSL_VERSION=1.2.5-r21

# We need to build dnsmasq from source due to this bug: https://thekelleys.org.uk/gitweb/?p=dnsmasq.git;a=commit;h=5846f749e5d878b6b5f7c20f6975bc96b95e4aae
# The bug is fixed in dnsmasq 2.92test11, but the latest release is 2.91.
# renovate: datasource=git-tags depName=https://thekelleys.org.uk/git/dnsmasq.git
ARG DNSMASQ_VERSION=v2.92test22

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

FROM alpine:3.23.0@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375

COPY --from=olsrd-builder /out /
COPY --from=babeld-builder /out /
COPY --from=ucode-builder /out /
COPY --from=dnsmasq-builder /out /

# renovate: datasource=repology depName=alpine_3_22/bash
ARG BASH_VERSION=5.3.3-r1
# renovate: datasource=repology depName=alpine_3_22/curl
ARG CURL_VERSION=8.17.0-r1
# renovate: datasource=repology depName=alpine_3_22/zlib
ARG ZLIB_VERSION=1.3.1-r2
# renovate: datasource=repology depName=alpine_3_22/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_22/openssl
ARG OPENSSL_VERSION=3.5.4-r0
# renovate: datasource=repology depName=alpine_3_22/iptables
ARG IPTABLES_VERSION=1.8.11-r1
# renovate: datasource=repology depName=alpine_3_22/iproute2
ARG IPRROUTE2_VERSION=6.17.0-r0
# renovate: datasource=repology depName=alpine_3_22/rsyslog
ARG RSYSLOG_VERSION=8.2506.0-r0
# renovate: datasource=repology depName=alpine_3_22/logrotate
ARG LOGROTATE_VERSION=3.22.0-r0
# renovate: datasource=repology depName=alpine_3_22/jq
ARG JQ_VERSION=1.8.1-r0
# renovate: datasource=repology depName=alpine_3_22/gettext
ARG GETTEXT_VERSION=0.24.1-r1
# renovate: datasource=repology depName=alpine_3_22/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20250521-r1
# renovate: datasource=repology depName=alpine_3_22/s6
ARG S6_VERSION=2.13.2.0-r0
# renovate: datasource=repology depName=alpine_3_22/cronie
ARG CRONIE_VERSION=1.7.2-r0
# renovate: datasource=repology depName=alpine_3_22/json-c
ARG JSONC_VERSION=0.18-r1
# renovate: datasource=repology depName=alpine_3_22/ca-certificates
ARG CA_CERTIFICATES_VERSION=20251003-r0
# renovate: datasource=repology depName=alpine_3_22/tcpdump
ARG TCPDUMP_VERSION=4.99.5-r1

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
    tcpdump="${TCPDUMP_VERSION}"

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

COPY --chown=root:root rootfs /
RUN chmod +x /etc/s6/*/run

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
