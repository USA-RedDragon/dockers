FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 AS builder

WORKDIR /tmp
RUN mkdir -p /out

COPY patches /patches

# renovate: datasource=repology depName=alpine_3_20/git
ARG GIT_VERSION=2.45.2-r0
# renovate: datasource=repology depName=alpine_3_20/gcc
ARG GCC_VERSION=13.2.1_git20240309-r0
# renovate: datasource=repology depName=alpine_3_20/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r0
# renovate: datasource=repology depName=alpine_3_20/bison
ARG BISON_VERSION=3.8.2-r1
# renovate: datasource=repology depName=alpine_3_20/flex
ARG FLEX_VERSION=2.6.4-r6
# renovate: datasource=repology depName=alpine_3_20/zlib
ARG ZLIB_VERSION=1.3.1-r1
# renovate: datasource=repology depName=alpine_3_20/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_20/binutils
ARG BINUTILS_VERSION=2.42-r0
# renovate: datasource=repology depName=alpine_3_20/openssl
ARG OPENSSL_VERSION=3.3.1-r3
# renovate: datasource=repology depName=alpine_3_20/curl
ARG CURL_VERSION=8.9.0-r0
# renovate: datasource=repology depName=alpine_3_20/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_20/musl
ARG MUSL_VERSION=1.2.5-r0
# renovate: datasource=repology depName=alpine_3_20/patch
ARG PATCH_VERSION=2.7.6-r10

RUN apk add --no-cache \
    git="${GIT_VERSION}" \
    gcc="${GCC_VERSION}" \
    linux-headers="${LINUX_HEADERS_VERSION}" \
    bison="${BISON_VERSION}" \
    flex="${FLEX_VERSION}" \
    zlib-dev="${ZLIB_VERSION}" \
    lzo-dev="${LZO_VERSION}" \
    binutils="${BINUTILS_VERSION}" \
    openssl-dev="${OPENSSL_VERSION}" \
    curl="${CURL_VERSION}" \
    make="${MAKE_VERSION}" \
    musl-dev="${MUSL_VERSION}" \
    patch="${PATCH_VERSION}"

# Build and install olsrd
RUN git clone https://github.com/USA-RedDragon/olsrd.git \
    && cd olsrd \
    && git checkout aredn \
    && make prefix=/usr \
    && make prefix=/usr DESTDIR=/out install arprefresh_install txtinfo_install jsoninfo_install dot_draw_install watchdog_install nameservice_install \
    && cd .. \
    && rm -rf olsrd

# Build and install vtun
RUN curl -fSsL https://downloads.sourceforge.net/project/vtun/vtun/3.0.3/vtun-3.0.3.tar.gz -o vtun-3.0.3.tar.gz \
    && tar -xzf vtun-3.0.3.tar.gz \
    && rm vtun-3.0.3.tar.gz \
    && cd vtun-3.0.3 \
    # --build=unknown-unknown-linux is magic for cross-compiling
    && ./configure --prefix=/usr --build=unknown-unknown-linux \
    && for patch in /patches/vtun/*.patch; do patch -p1 < "$patch"; done \
    && make \
    && make DESTDIR=/out install \
    && cd .. \
    && rm -rf vtun-3.0.3

FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5

COPY --from=builder /out /

# renovate: datasource=repology depName=alpine_3_20/bash
ARG BASH_VERSION=5.2.26-r0
# renovate: datasource=repology depName=alpine_3_20/curl
ARG CURL_VERSION=8.9.0-r0
# renovate: datasource=repology depName=alpine_3_20/zlib
ARG ZLIB_VERSION=1.3.1-r1
# renovate: datasource=repology depName=alpine_3_20/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_20/openssl
ARG OPENSSL_VERSION=3.3.1-r3
# renovate: datasource=repology depName=alpine_3_20/iptables
ARG IPTABLES_VERSION=1.8.10-r3
# renovate: datasource=repology depName=alpine_3_20/iproute2
ARG IPRROUTE2_VERSION=6.9.0-r0
# renovate: datasource=repology depName=alpine_3_20/rsyslog
ARG RSYSLOG_VERSION=8.2404.0-r0
# renovate: datasource=repology depName=alpine_3_20/logrotate
ARG LOGROTATE_VERSION=3.21.0-r1
# renovate: datasource=repology depName=alpine_3_20/dnsmasq
ARG DNSMASQ_VERSION=2.90-r3
# renovate: datasource=repology depName=alpine_3_20/jq
ARG JQ_VERSION=1.7.1-r0
# renovate: datasource=repology depName=alpine_3_20/gettext
ARG GETTEXT_VERSION=0.22.5-r0
# renovate: datasource=repology depName=alpine_3_20/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r4
# renovate: datasource=repology depName=alpine_3_20/s6
ARG S6_VERSION=2.12.0.4-r0
# renovate: datasource=repology depName=alpine_3_20/cronie
ARG CRONIE_VERSION=1.7.2-r0

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
    dnsmasq="${DNSMASQ_VERSION}" \
    jq="${JQ_VERSION}" \
    gettext="${GETTEXT_VERSION}" \
    wireguard-tools="${WIREGUARD_TOOLS_VERSION}" \
    s6="${S6_VERSION}" \
    cronie="${CRONIE_VERSION}"

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

COPY --chown=root:root rootfs /
RUN chmod +x /etc/s6/*/run

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
