FROM debian:12.15-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8

# renovate: datasource=git-tags extractVersion=^v(?<version>.*)$ depName=https://github.com/FoldingAtHome/fah-client-bastet
ARG FOLDINGATHOME_VERSION=8.5.6
# renovate: datasource=git-tags extractVersion=^bastet-v(?<version>.*)$ depName=https://github.com/cauldrondevelopmentllc/cbang
ARG CBANG_VERSION=8.5.6

ARG DEV_PACKAGES=" \
    libssl-dev \
    zlib1g-dev \
    libbz2-dev \
    liblz4-dev \
    libsystemd-dev \
"

ARG BUILD_DEPS=" \
    build-essential \
    scons \
    git \
    npm \
    fakeroot \
"

# Runtime dependencies
RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        zlib1g \
        ca-certificates \
        bzip2 \
        openssl \
        liblz4-1 \
        systemd \
        udev \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ${BUILD_DEPS} \
        ${DEV_PACKAGES} \
    && git clone --single-branch --branch v${FOLDINGATHOME_VERSION} https://github.com/FoldingAtHome/fah-client-bastet /tmp/fah-client-bastet \
    && git clone --single-branch --branch bastet-v${CBANG_VERSION} https://github.com/cauldrondevelopmentllc/cbang /tmp/cbang \
    && cd /tmp \
    && export CBANG_HOME=/tmp/cbang \
    && scons -C cbang \
    && scons -C fah-client-bastet \
    && scons -C fah-client-bastet package \
    && apt-get purge -y --auto-remove \
        ${BUILD_DEPS} \
        ${DEV_PACKAGES} \
    && apt-get install --no-install-recommends -y ./fah-client-bastet/fah-client_${FOLDINGATHOME_VERSION}_*.deb \
    && cd / \
    && rm -rf /tmp/fah-client-bastet \
    && rm -rf /tmp/cbang \
    && rm -rf /var/lib/apt/lists/*

USER fah-client
WORKDIR /var/lib/fah-client

ENTRYPOINT ["/usr/bin/fah-client"]
CMD ["--config=/etc/fah-client/config.xml", "--log=/var/log/fah-client/log.txt", "--log-rotate-dir=/var/log/fah-client/"]
