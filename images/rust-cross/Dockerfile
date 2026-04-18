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
    'DL_CMD = curl -fSL --retry 50 --retry-delay 5 --retry-all-errors -o' \
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

FROM debian:12.13-slim@sha256:4724b8cc51e33e398f0e2e15e18d5ec2851ff0c2280647e1310bc1642182655d AS build-osxcross

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake curl git patch \
    liblzma-dev libxml2-dev libssl-dev zlib1g-dev \
    python3 git bzip2 ca-certificates \
    gnupg lsb-release software-properties-common wget \
    && wget -qO- https://apt.llvm.org/llvm-snapshot.gpg.key | tee /etc/apt/trusted.gpg.d/apt.llvm.org.asc \
    && echo "deb http://apt.llvm.org/bookworm/ llvm-toolchain-bookworm-19 main" > /etc/apt/sources.list.d/llvm.list \
    && apt-get update && apt-get install -y --no-install-recommends clang-19 \
    && ln -sf /usr/bin/clang-19 /usr/bin/clang && ln -sf /usr/bin/clang++-19 /usr/bin/clang++ \
    && rm -rf /var/lib/apt/lists/*

# renovate: sha: datasource=git-refs depName=osxcross packageName=https://github.com/tpoechtrager/osxcross branch=master
ARG OSXCROSS_SHA=e6ab3fa7423f9235ce9ed6381d6d3af191b46b59

ARG DARWIN_SDK_VERSION=26.4
ARG DARWIN_OSX_VERSION_MIN=10.13

RUN --mount=type=secret,id=DARWIN_SDK_URL \
    mkdir -p /tmp/osxcross && cd /tmp/osxcross \
    && git init . \
    && git remote add origin https://github.com/tpoechtrager/osxcross.git \
    && git fetch --depth 1 origin ${OSXCROSS_SHA} \
    && git checkout FETCH_HEAD \
    && curl -sLo tarballs/MacOSX${DARWIN_SDK_VERSION}.sdk.tar.bz2 "$(cat /run/secrets/DARWIN_SDK_URL)" \
    && sed -i '/\*) echo "Unsupported SDK"/i\  26.4*) TARGET=darwin25.4;   SUPPORTED_ARCHS="arm64 arm64e x86_64 x86_64h"; NEED_TAPI_SUPPORT=1; OSX_VERSION_MIN_INT=10.13 ;;' build.sh \
    && UNATTENDED=1 SDK_VERSION="${DARWIN_SDK_VERSION}" OSX_VERSION_MIN="${DARWIN_OSX_VERSION_MIN}" ./build.sh \
    && mv target /opt/osxcross && mv tools /opt/osxcross/ \
    && rm -rf /tmp/osxcross

FROM rust:1.95.0-slim-bookworm@sha256:5cad7981952984a5c222b6715e1ee282f9bd98ffc53e44e069f7153d49293104

COPY --from=build-mips /opt/cross/ /opt/cross/
COPY --from=build-mips-sf /opt/cross/ /opt/cross/
COPY --from=build-mipsel /opt/cross/ /opt/cross/
COPY --from=build-mipsel-sf /opt/cross/ /opt/cross/
COPY --from=build-arm /opt/cross/ /opt/cross/
COPY --from=build-aarch64 /opt/cross/ /opt/cross/
COPY --from=build-x86_64 /opt/cross/ /opt/cross/
COPY --from=build-riscv64 /opt/cross/ /opt/cross/
COPY --from=build-osxcross /opt/osxcross/ /opt/osxcross/

ARG DARWIN_SDK_VERSION=25.4
ENV PATH="/opt/cross/bin:/opt/osxcross/bin:${PATH}" \
    LD_LIBRARY_PATH="/opt/osxcross/lib" \
    CARGO_TARGET_X86_64_PC_WINDOWS_GNU_LINKER=x86_64-w64-mingw32-gcc \
    CARGO_TARGET_X86_64_APPLE_DARWIN_LINKER=x86_64-apple-darwin${DARWIN_SDK_VERSION}-clang \
    CARGO_TARGET_AARCH64_APPLE_DARWIN_LINKER=arm64-apple-darwin${DARWIN_SDK_VERSION}-clang

# Tier 1 / Tier 2 targets — pre-built std available on stable
RUN rustup target add \
    armv7-unknown-linux-musleabihf \
    aarch64-unknown-linux-musl \
    x86_64-unknown-linux-musl \
    riscv64gc-unknown-linux-musl \
    x86_64-pc-windows-gnu \
    x86_64-apple-darwin \
    aarch64-apple-darwin

# Tier 3 targets (MIPS) — no pre-built std, requires nightly + build-std
RUN rustup toolchain install nightly && \
    rustup component add rust-src --toolchain nightly

RUN apt-get update && \
    apt-get install -y \
        --no-install-recommends \
        --no-install-suggests \
            mingw-w64 \
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
