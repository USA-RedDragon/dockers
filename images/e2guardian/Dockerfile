FROM debian:13.7-slim@sha256:a99cfc517144bc59b1978475ec53b46ecabec7e43635402ee5b77cc54cd1b20a

# renovate: datasource=github-releases depName=e2guardian/e2guardian
ARG E2GUARDIAN_VERSION=v5.5.4r

# Build dependencies

ARG DEV_PACKAGES=" \
    libpcre3-dev \
    libssl-dev \
    zlib1g-dev \
    libclamav-dev \
"

ARG BUILD_DEPS=" \
    make \
    g++ \
    git \
    automake \
    autoconf \
    pkg-config \
    m4 \
"

# Runtime dependencies
RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        zlib1g \
        ca-certificates \
        libpcre3 \
        openssl \
        clamav \
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
