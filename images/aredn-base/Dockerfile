FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS olsrd-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_21/git
ARG GIT_VERSION=2.47.2-r0
# renovate: datasource=repology depName=alpine_3_21/gcc
ARG GCC_VERSION=14.2.0-r4
# renovate: datasource=repology depName=alpine_3_21/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r1
# renovate: datasource=repology depName=alpine_3_21/bison
ARG BISON_VERSION=3.8.2-r1
# renovate: datasource=repology depName=alpine_3_21/flex
ARG FLEX_VERSION=2.6.4-r6
# renovate: datasource=repology depName=alpine_3_21/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_21/musl
ARG MUSL_VERSION=1.2.5-r9

ARG OLSRD_VERSION=6a77e736fb95392e2088f5e6a70f2cb1ce6e1603

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

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS babeld-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_21/git
ARG GIT_VERSION=2.47.2-r0
# renovate: datasource=repology depName=alpine_3_21/gcc
ARG GCC_VERSION=14.2.0-r4
# renovate: datasource=repology depName=alpine_3_21/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r1
# renovate: datasource=repology depName=alpine_3_21/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_21/musl
ARG MUSL_VERSION=1.2.5-r9

ARG BABELD_VERSION=c21fe609f3fc2a199126ea1cd55733fb9399c956

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

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS ucode-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_21/git
ARG GIT_VERSION=2.47.2-r0
# renovate: datasource=repology depName=alpine_3_21/gcc
ARG GCC_VERSION=14.2.0-r4
# renovate: datasource=repology depName=alpine_3_21/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r1
# renovate: datasource=repology depName=alpine_3_21/zlib
ARG ZLIB_VERSION=1.3.1-r2
# renovate: datasource=repology depName=alpine_3_21/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_21/musl
ARG MUSL_VERSION=1.2.5-r9
# renovate: datasource=repology depName=alpine_3_21/cmake
ARG CMAKE_VERSION=3.31.1-r0
# renovate: datasource=repology depName=alpine_3_21/json-c
ARG JSONC_VERSION=0.18-r0

ARG UCODE_VERSION=37ac8f112af63e64c1ea2d13ae63c134b62a5681

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

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS dnsmasq-builder

WORKDIR /tmp
RUN mkdir -p /out

# renovate: datasource=repology depName=alpine_3_21/git
ARG GIT_VERSION=2.47.2-r0
# renovate: datasource=repology depName=alpine_3_21/gcc
ARG GCC_VERSION=14.2.0-r4
# renovate: datasource=repology depName=alpine_3_21/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r1
# renovate: datasource=repology depName=alpine_3_21/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_21/musl
ARG MUSL_VERSION=1.2.5-r9

# We need to build dnsmasq from source due to this bug: https://thekelleys.org.uk/gitweb/?p=dnsmasq.git;a=commit;h=5846f749e5d878b6b5f7c20f6975bc96b95e4aae
# The bug is fixed in dnsmasq 2.92test11, but the latest release is 2.91.
# renovate: datasource=git-tags depName=https://thekelleys.org.uk/git/dnsmasq.git
ARG DNSMASQ_VERSION=v2.92test11

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

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c

COPY --from=olsrd-builder /out /
COPY --from=babeld-builder /out /
COPY --from=ucode-builder /out /
COPY --from=dnsmasq-builder /out /

# renovate: datasource=repology depName=alpine_3_21/bash
ARG BASH_VERSION=5.2.37-r0
# renovate: datasource=repology depName=alpine_3_21/curl
ARG CURL_VERSION=8.12.1-r1
# renovate: datasource=repology depName=alpine_3_21/zlib
ARG ZLIB_VERSION=1.3.1-r2
# renovate: datasource=repology depName=alpine_3_21/lzo
ARG LZO_VERSION=2.10-r5
# renovate: datasource=repology depName=alpine_3_21/openssl
ARG OPENSSL_VERSION=3.3.3-r0
# renovate: datasource=repology depName=alpine_3_21/iptables
ARG IPTABLES_VERSION=1.8.11-r1
# renovate: datasource=repology depName=alpine_3_21/iproute2
ARG IPRROUTE2_VERSION=6.11.0-r0
# renovate: datasource=repology depName=alpine_3_21/rsyslog
ARG RSYSLOG_VERSION=8.2410.0-r0
# renovate: datasource=repology depName=alpine_3_21/logrotate
ARG LOGROTATE_VERSION=3.21.0-r1
# renovate: datasource=repology depName=alpine_3_21/jq
ARG JQ_VERSION=1.7.1-r0
# renovate: datasource=repology depName=alpine_3_21/gettext
ARG GETTEXT_VERSION=0.22.5-r0
# renovate: datasource=repology depName=alpine_3_21/wireguard-tools
ARG WIREGUARD_TOOLS_VERSION=1.0.20210914-r4
# renovate: datasource=repology depName=alpine_3_21/s6
ARG S6_VERSION=2.13.1.0-r0
# renovate: datasource=repology depName=alpine_3_21/cronie
ARG CRONIE_VERSION=1.7.2-r0
# renovate: datasource=repology depName=alpine_3_21/json-c
ARG JSONC_VERSION=0.18-r0
# renovate: datasource=repology depName=alpine_3_21/ca-certificates
ARG CA_CERTIFICATES_VERSION=20241121-r1

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
    ca-certificates="${CA_CERTIFICATES_VERSION}"

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

COPY --chown=root:root rootfs /
RUN chmod +x /etc/s6/*/run

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
