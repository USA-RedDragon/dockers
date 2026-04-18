FROM debian:12.13-slim@sha256:4724b8cc51e33e398f0e2e15e18d5ec2851ff0c2280647e1310bc1642182655d AS toolchain-base

RUN apt-get update && apt-get install -y \
    build-essential file git curl wget patch

# renovate: datasource=git-tags depName=https://github.com/richfelker/musl-cross-make
ARG MUSL_CROSS_MAKE_VERSION=v0.9.11

RUN git clone --single-branch --branch ${MUSL_CROSS_MAKE_VERSION} --depth 1 https://github.com/richfelker/musl-cross-make.git /build/musl-cross-make
WORKDIR /build/musl-cross-make

RUN printf '%s\n' \
    'COMMON_CONFIG += --disable-nls' \
    'GCC_CONFIG += --disable-libquadmath --disable-decimal-float' \
    'COMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"' \
    'MUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"' \
    'MUSL_CONFIG += CFLAGS="-fPIC -O2"' \
    'GNU_SITE = https://ftpmirror.gnu.org/gnu' \
    'DL_CMD = curl -fSL --retry 5 --retry-delay 3 --retry-all-errors -o' \
    'OUTPUT = /opt/cross' \
    > config.mak.base

# Each toolchain builds in its own stage so BuildKit can run them in parallel.

FROM toolchain-base AS build-mips
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = mips-linux-musl' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-mips-sf
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = mips-linux-muslsf' 'GCC_CONFIG += --with-float=soft' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-mipsel
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = mipsel-linux-musl' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-mipsel-sf
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = mipsel-linux-muslsf' 'GCC_CONFIG += --with-float=soft' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-arm
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = arm-linux-musleabihf' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-aarch64
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = aarch64-linux-musl' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-x86_64
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = x86_64-linux-musl' >> config.mak && \
    make -j$(nproc) install

FROM toolchain-base AS build-riscv64
RUN cp config.mak.base config.mak && \
    printf '%s\n' 'TARGET = riscv64-linux-musl' >> config.mak && \
    make -j$(nproc) install

FROM rust:1.94.1-slim-bookworm@sha256:cf9dd0ec73e75f827fe59123fff9dc65af1a1c8363c3c31ee8d7f8ad0b6a5fb2

COPY --from=build-mips /opt/cross/ /opt/cross/
COPY --from=build-mips-sf /opt/cross/ /opt/cross/
COPY --from=build-mipsel /opt/cross/ /opt/cross/
COPY --from=build-mipsel-sf /opt/cross/ /opt/cross/
COPY --from=build-arm /opt/cross/ /opt/cross/
COPY --from=build-aarch64 /opt/cross/ /opt/cross/
COPY --from=build-x86_64 /opt/cross/ /opt/cross/
COPY --from=build-riscv64 /opt/cross/ /opt/cross/
ENV PATH="/opt/cross/bin:${PATH}"

# Tier 2 targets — pre-built std available on stable
RUN rustup target add \
    armv7-unknown-linux-musleabihf \
    aarch64-unknown-linux-musl \
    x86_64-unknown-linux-musl \
    riscv64gc-unknown-linux-musl

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
