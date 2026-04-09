FROM debian:13.4-slim@sha256:4ffb3a1511099754cddc70eb1b12e50ffdb67619aa0ab6c13fcd800a78ef7c7a AS toolchain-builder

RUN apt-get update && apt-get install -y \
    build-essential file git curl wget patch

# renovate: datasource=git-tags depName=https://github.com/richfelker/musl-cross-make
ARG MUSL_CROSS_MAKE_VERSION=v0.9.11

RUN git clone --single-branch --branch ${MUSL_CROSS_MAKE_VERSION} --depth 1 https://github.com/richfelker/musl-cross-make.git /build/musl-cross-make
WORKDIR /build/musl-cross-make

RUN printf 'TARGET = %s\nGCC_CONFIG = %s\nCOMMON_CONFIG += --disable-nls\nGCC_CONFIG += --disable-libquadmath --disable-decimal-float\nCOMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-fPIC -O2"\nGNU_SITE = https://ftpmirror.gnu.org/gnu\nDL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o\nOUTPUT = /opt/cross\n' \
    "mips-linux-muslsf" "--with-float=soft" > config.mak && make -j$(nproc) install

RUN printf 'TARGET = %s\nGCC_CONFIG = %s\nCOMMON_CONFIG += --disable-nls\nGCC_CONFIG += --disable-libquadmath --disable-decimal-float\nCOMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-fPIC -O2"\nGNU_SITE = https://ftpmirror.gnu.org/gnu\nDL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o\nOUTPUT = /opt/cross\n' \
    "mipsel-linux-muslsf" "--with-float=soft" > config.mak && make -j$(nproc) install

RUN printf 'TARGET = %s\nGCC_CONFIG = %s\nCOMMON_CONFIG += --disable-nls\nGCC_CONFIG += --disable-libquadmath --disable-decimal-float\nCOMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-fPIC -O2"\nGNU_SITE = https://ftpmirror.gnu.org/gnu\nDL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o\nOUTPUT = /opt/cross\n' \
    "arm-linux-musleabihf" "--with-arch=armv7-a --with-fpu=neon-vfpv4 --with-float=hard" > config.mak && make -j$(nproc) install

RUN printf 'TARGET = %s\nGCC_CONFIG = %s\nCOMMON_CONFIG += --disable-nls\nGCC_CONFIG += --disable-libquadmath --disable-decimal-float\nCOMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-fPIC -O2"\nGNU_SITE = https://ftpmirror.gnu.org/gnu\nDL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o\nOUTPUT = /opt/cross\n' \
    "aarch64-linux-musl" "" > config.mak && make -j$(nproc) install

RUN printf 'TARGET = %s\nGCC_CONFIG = %s\nCOMMON_CONFIG += --disable-nls\nGCC_CONFIG += --disable-libquadmath --disable-decimal-float\nCOMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"\nMUSL_CONFIG += CFLAGS="-fPIC -O2"\nGNU_SITE = https://ftpmirror.gnu.org/gnu\nDL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o\nOUTPUT = /opt/cross\n' \
    "x86_64-linux-musl" "" > config.mak && make -j$(nproc) install

FROM rust:1.94.1-slim-bookworm@sha256:cf9dd0ec73e75f827fe59123fff9dc65af1a1c8363c3c31ee8d7f8ad0b6a5fb2

COPY --from=toolchain-builder /opt/cross /opt/cross
ENV PATH="/opt/cross/bin:${PATH}"

# Tier 2 targets — pre-built std available on stable
RUN rustup target add \
    armv7-unknown-linux-musleabihf \
    aarch64-unknown-linux-musl \
    x86_64-unknown-linux-musl

# Tier 3 targets (MIPS) — no pre-built std, requires nightly + build-std
RUN rustup toolchain install nightly && \
    rustup component add rust-src --toolchain nightly

RUN apt-get update && \
    apt-get install -y \
        --no-install-recommends \
        --no-install-suggests \
            qemu-user-static \
    && \
    rm -rf /var/lib/apt/lists/*

# renovate: datasource=github-releases depName=protocolbuffers/protobuf
ARG PROTOBUF_VERSION=v34.1
ARG TARGETARCH

RUN apt-get update && \
    apt-get install -y \
        --no-install-recommends \
        --no-install-suggests \
            curl \
            unzip \
    && \
    ARCH="${TARGETARCH}" && \
    if [ "${TARGETARCH}" = "arm64" ]; then ARCH="aarch_64"; fi && \
    if [ "${TARGETARCH}" = "amd64" ]; then ARCH="x86_64"; fi && \
    curl -sSLo /tmp/protobuf.zip https://github.com/protocolbuffers/protobuf/releases/download/${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION#v}-linux-${ARCH}.zip && \
    unzip -q /tmp/protobuf.zip -d /usr/local bin/protoc && \
    chmod +x /usr/local/bin/protoc && \
    rm /tmp/protobuf.zip && \
    apt-get -y remove --purge \
        curl \
        unzip \
    && apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
