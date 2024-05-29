FROM alpine:3.20.0@sha256:216266c86fc4dcef5619930bd394245824c2af52fd21ba7c6fa0e618657d4c3b as builder

WORKDIR /tmp
RUN mkdir -p /out

COPY patches /patches

# renovate: datasource=repology depName=alpine_3_19/git
ARG GIT_VERSION=2.43.0-r0
# renovate: datasource=repology depName=alpine_3_19/gcc
ARG GCC_VERSION=13.2.1_git20231014-r0
# renovate: datasource=repology depName=alpine_3_19/linux-headers
ARG LINUX_HEADERS_VERSION=6.5-r0
# renovate: datasource=repology depName=alpine_3_19/bison
ARG BISON_VERSION=3.8.2-r1
# renovate: datasource=repology depName=alpine_3_19/flex
ARG FLEX_VERSION=2.6.4-r6
# renovate: datasource=repology depName=alpine_3_19/zlib
ARG ZLIB_VERSION=1.3.1-r0
# renovate: datasource=repology depName=alpine_3_19/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_19/binutils
ARG BINUTILS_VERSION=2.41-r0
# renovate: datasource=repology depName=alpine_3_19/openssl
ARG OPENSSL_VERSION=3.1.4-r6
# renovate: datasource=repology depName=alpine_3_19/curl
ARG CURL_VERSION=8.5.0-r0
# renovate: datasource=repology depName=alpine_3_19/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_19/musl
ARG MUSL_VERSION=1.2.4_git20230717-r4
# renovate: datasource=repology depName=alpine_3_19/patch
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

FROM alpine:3.20.0@sha256:216266c86fc4dcef5619930bd394245824c2af52fd21ba7c6fa0e618657d4c3b

COPY --from=builder /out /

# renovate: datasource=repology depName=alpine_3_19/bash
ARG BASH_VERSION=5.2.21-r0
# renovate: datasource=repology depName=alpine_3_19/curl
ARG CURL_VERSION=8.5.0-r0
# renovate: datasource=repology depName=alpine_3_19/zlib
ARG ZLIB_VERSION=1.3.1-r0
# renovate: datasource=repology depName=alpine_3_19/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_19/openssl
ARG OPENSSL_VERSION=3.1.4-r6
# renovate: datasource=repology depName=alpine_3_19/iptables
ARG IPTABLES_VERSION=1.8.10-r3
# renovate: datasource=repology depName=alpine_3_19/iproute2
ARG IPRROUTE2_VERSION=6.6.0-r0
# renovate: datasource=repology depName=alpine_3_19/rsyslog
ARG RSYSLOG_VERSION=8.2310.0-r0
# renovate: datasource=repology depName=alpine_3_19/logrotate
ARG LOGROTATE_VERSION=3.21.0-r1
# renovate: datasource=repology depName=alpine_3_19/dnsmasq
ARG DNSMASQ_VERSION=2.90-r2
# renovate: datasource=repology depName=alpine_3_19/jq
ARG JQ_VERSION=1.7.1-r0
# renovate: datasource=repology depName=alpine_3_19/gettext
ARG GETTEXT_VERSION=0.22.3-r0
# renovate: datasource=repology depName=alpine_3_19/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r3
# renovate: datasource=repology depName=alpine_3_19/s6
ARG S6_VERSION=2.12.0.2-r0
# renovate: datasource=repology depName=alpine_3_19/cronie
ARG CRONIE_VERSION=1.7.0-r0

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
