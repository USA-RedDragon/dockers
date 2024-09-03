FROM debian:12.6-slim@sha256:2ccc7e39b0a6f504d252f807da1fc4b5bcd838e83e4dec3e2f57b2a4a64e7214

# renovate: datasource=github-releases depName=e2guardian/e2guardian
ARG E2GUARDIAN_VERSION=v5.5.4r

# Runtime dependencies

# renovate: datasource=repology versioning=deb depName=debian_12/zlib
ARG ZLIB1G_VERSION=1:1.2.13.dfsg-1
# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ARG CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/pcre3
ARG LIBPCRE3_VERSION=2:8.39-15
# renovate: datasource=repology versioning=deb depName=debian_12/openssl
ARG OPENSSL_VERSION=3.0.14-1~deb12u2
# renovate: datasource=repology versioning=deb depName=debian_12/clamav
ARG CLAMAV_VERSION=1.0.5+dfsg-1~deb12u1

# Build dependencies

ARG DEV_PACKAGES=" \
    libpcre3-dev=${LIBPCRE3_VERSION} \
    libssl-dev=${OPENSSL_VERSION} \
    zlib1g-dev=${ZLIB1G_VERSION} \
    libclamav-dev=${CLAMAV_VERSION} \
"

# renovate: datasource=repology versioning=deb depName=debian_12/make-dfsg
ARG MAKE_VERSION=4.3-4.1
# renovate: datasource=repology versioning=deb depName=debian_12/gcc
ARG GCC_VERSION=4:12.2.0-3
# renovate: datasource=repology versioning=deb depName=debian_12/git
ARG GIT_VERSION=1:2.39.2-1.1
# renovate: datasource=repology versioning=deb depName=debian_12/automake-1.16
ARG AUTOMAKE_VERSION=1:1.16.5-1.3
# renovate: datasource=repology versioning=deb depName=debian_12/autoconf
ARG AUTOCONF_VERSION=2.71-3
# renovate: datasource=repology versioning=deb depName=debian_12/pkgconf
ARG PKG_CONFIG_VERSION=1.8.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/m4
ARG M4_VERSION=1.4.19-3

ARG BUILD_DEPS=" \
    make=${MAKE_VERSION} \
    g++=${GCC_VERSION} \
    git=${GIT_VERSION} \
    automake=${AUTOMAKE_VERSION} \
    autoconf=${AUTOCONF_VERSION} \
    pkg-config=${PKG_CONFIG_VERSION} \
    m4=${M4_VERSION} \
"

# Runtime dependencies
RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        zlib1g="${ZLIB1G_VERSION}" \
        ca-certificates="${CA_CERTIFICATES_VERSION}" \
        libpcre3="${LIBPCRE3_VERSION}" \
        openssl="${OPENSSL_VERSION}" \
        clamav="${CLAMAV_VERSION}" \
    && rm -rf /var/lib/apt/lists/*

# False-positive from hadolint
# hadolint ignore=DL3008
RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ${BUILD_DEPS} \
        ${DEV_PACKAGES} \
    && git clone --single-branch --branch ${E2GUARDIAN_VERSION} https://github.com/e2guardian/e2guardian.git /tmp/e2guardian \
    && cd /tmp/e2guardian \
    # Fix -N option
    && git config --global user.email "git@example.com" \
    && git config --global user.name "git" \
    && git fetch origin v5.5 \
    && git cherry-pick eddc75208b48012c9355ee9f3026d50a5a857c30 \
    && ./autogen.sh \
    && ./configure \
        --prefix=/usr \
        --sbindir=/usr/bin \
        --sysconfdir=/etc \
        --localstatedir=/var \
        --with-piddir=/var/run/e2guardian \
        --enable-clamd=yes \
        --enable-commandline=yes \
        --enable-email=yes \
        --enable-pcre=yes \
        --enable-sslmitm=yes \
        --with-proxygroup=e2guardian \
        --with-proxyuser=e2guardian \
        CPPFLAGS=-O2 \
    && make -j"$(nproc)" \
    && make install \
    && apt-get remove --purge -y \
        ${BUILD_DEPS} \
        ${DEV_PACKAGES} \
    && cd / \
    && rm -rf /tmp/e2guardian \
    && rm -rf /var/lib/apt/lists/*

RUN sed -i 's/#dockermode = off/dockermode = on/' /etc/e2guardian/e2guardian.conf

RUN groupadd -r e2guardian \
    && useradd -r -g e2guardian -d /e2guardian -s /sbin/nologin -c "e2guardian user" e2guardian \
    && mkdir -p /var/run/e2guardian \
    && chown -R e2guardian:e2guardian /var/run/e2guardian

USER e2guardian:e2guardian

EXPOSE 8080
EXPOSE 8443

CMD ["/usr/bin/e2guardian", "-N"]
