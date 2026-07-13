FROM debian:12.13-slim@sha256:4724b8cc51e33e398f0e2e15e18d5ec2851ff0c2280647e1310bc1642182655d AS toolchain-base

RUN apt-get update && apt-get install -y \
    build-essential file git curl wget patch

# renovate: datasource=git-tags depName=https://github.com/richfelker/musl-cross-make
ARG MUSL_CROSS_MAKE_VERSION=v0.9.11

RUN git clone --single-branch --branch ${MUSL_CROSS_MAKE_VERSION} --depth 1 https://github.com/richfelker/musl-cross-make.git /build/musl-cross-make
WORKDIR /build/musl-cross-make

# git.savannah.gnu.org (musl-cross-make's config.sub source) is frequently down.
# Pre-seed the EXACT pinned revision (3d5db9ebe860) from Software Heritage, which
# is content-addressed by the same sha1 musl-cross-make verifies. Because the file
# then already exists, make's download rule (an order-only prereq) never runs, so
# savannah is never contacted — but it's byte-identical to the pin, not a rolling
# gcc config.sub. The sha1 check keeps it honest if SWH ever changes.
RUN mkdir -p sources && \
    curl -fSL --retry 5 -o sources/config.sub \
      "https://archive.softwareheritage.org/api/1/content/sha1:7d66e32bb3cce017e1cc9bef59fb6f8271fb7fec/raw/" && \
    echo "7d66e32bb3cce017e1cc9bef59fb6f8271fb7fec  sources/config.sub" | sha1sum -c -

RUN printf '%s\n' \
    'COMMON_CONFIG += --disable-nls' \
    'GCC_CONFIG += --disable-libquadmath --disable-decimal-float' \
    'COMMON_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"' \
    'MUSL_CONFIG += CFLAGS="-g0 -O2" CXXFLAGS="-g0 -O2"' \
    'MUSL_CONFIG += CFLAGS="-fPIC -O2"' \
    'GNU_SITE = https://mirrors.kernel.org/gnu' \
    'DL_CMD = curl -fSL --connect-timeout 30 --retry 5 --retry-delay 5 --retry-all-errors -o' \
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

FROM rust:1.94.1-slim-bookworm@sha256:cf9dd0ec73e75f827fe59123fff9dc65af1a1c8363c3c31ee8d7f8ad0b6a5fb2

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
ENV PATH="/opt/cross/bin:/opt/osxcross/bin:/opt/llvm-mingw/bin:${PATH}" \
    LD_LIBRARY_PATH="/opt/osxcross/lib" \
    CARGO_TARGET_X86_64_PC_WINDOWS_GNULLVM_LINKER=x86_64-w64-mingw32-clang \
    CARGO_TARGET_AARCH64_PC_WINDOWS_GNULLVM_LINKER=aarch64-w64-mingw32-clang \
    CARGO_TARGET_X86_64_APPLE_DARWIN_LINKER=x86_64-apple-darwin${DARWIN_SDK_VERSION}-clang \
    CARGO_TARGET_AARCH64_APPLE_DARWIN_LINKER=arm64-apple-darwin${DARWIN_SDK_VERSION}-clang

# Tier 1 / Tier 2 targets — pre-built std available on stable
RUN rustup target add \
    armv7-unknown-linux-musleabihf \
    aarch64-unknown-linux-musl \
    x86_64-unknown-linux-musl \
    riscv64gc-unknown-linux-gnu \
    riscv64gc-unknown-linux-musl \
    x86_64-pc-windows-gnullvm \
    aarch64-pc-windows-gnullvm \
    x86_64-apple-darwin \
    aarch64-apple-darwin

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

# LLVM/Clang-based MinGW toolchain, providing the *-w64-mingw32-clang linkers
# used by the x86_64/aarch64-pc-windows-gnullvm targets. One toolchain targets
# both Windows arches; extracted to a stable /opt/llvm-mingw on PATH.
# renovate: datasource=github-releases depName=mstorsjo/llvm-mingw
ARG LLVM_MINGW_VERSION=20260616
ARG TARGETARCH
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl xz-utils && \
    ARCH="${TARGETARCH}" && \
    if [ "${TARGETARCH}" = "arm64" ]; then ARCH="aarch64"; fi && \
    if [ "${TARGETARCH}" = "amd64" ]; then ARCH="x86_64"; fi && \
    curl -fSL -o /tmp/llvm-mingw.tar.xz \
        "https://github.com/mstorsjo/llvm-mingw/releases/download/${LLVM_MINGW_VERSION}/llvm-mingw-${LLVM_MINGW_VERSION}-ucrt-ubuntu-22.04-${ARCH}.tar.xz" && \
    mkdir -p /opt/llvm-mingw && \
    tar -xJf /tmp/llvm-mingw.tar.xz -C /opt/llvm-mingw --strip-components=1 && \
    apt-get -y remove --purge curl xz-utils && apt-get -y autoremove && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

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

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates gnupg wget \
    && wget -qO- https://apt.llvm.org/llvm-snapshot.gpg.key | tee /etc/apt/trusted.gpg.d/apt.llvm.org.asc >/dev/null \
    && echo "deb http://apt.llvm.org/bookworm/ llvm-toolchain-bookworm-19 main" > /etc/apt/sources.list.d/llvm.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends libclang-19-dev clang-19 \
    && ln -sf /usr/bin/clang-19 /usr/bin/clang \
    && ln -sf /usr/bin/clang++-19 /usr/bin/clang++ \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV LIBCLANG_PATH=/usr/lib/llvm-19/lib \
    BINDGEN_EXTRA_CLANG_ARGS="-isystem /usr/lib/llvm-19/lib/clang/19/include"

RUN dpkg --add-architecture arm64 && \
    dpkg --add-architecture armhf && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        openjdk-17-jdk-headless \
        pkg-config \
        gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
        gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf \
        gcc-riscv64-linux-gnu g++-riscv64-linux-gnu \
        libasound2-dev libx11-dev libxcursor-dev libxrandr-dev libxi-dev \
        libwayland-dev libxkbcommon-dev libudev-dev \
        libasound2-dev:arm64 libx11-dev:arm64 libxcursor-dev:arm64 \
        libxrandr-dev:arm64 libxi-dev:arm64 libwayland-dev:arm64 \
        libxkbcommon-dev:arm64 libudev-dev:arm64 \
        libasound2-dev:armhf libx11-dev:armhf libxcursor-dev:armhf \
        libxrandr-dev:armhf libxi-dev:armhf libwayland-dev:armhf \
        libxkbcommon-dev:armhf libudev-dev:armhf \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
ENV JAVA_HOME=/usr/lib/jvm/java-17-openjdk-${TARGETARCH}

RUN rustup target add \
    x86_64-unknown-linux-gnu \
    aarch64-unknown-linux-gnu \
    armv7-unknown-linux-gnueabihf \
    wasm32-unknown-unknown \
    aarch64-linux-android \
    armv7-linux-androideabi \
    x86_64-linux-android \
    i686-linux-android

ENV CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER=x86_64-linux-musl-gcc \
    CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER=aarch64-linux-musl-gcc \
    CARGO_TARGET_ARMV7_UNKNOWN_LINUX_MUSLEABIHF_LINKER=arm-linux-musleabihf-gcc \
    CARGO_TARGET_RISCV64GC_UNKNOWN_LINUX_MUSL_LINKER=riscv64-linux-musl-gcc

ENV CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc \
    CARGO_TARGET_ARMV7_UNKNOWN_LINUX_GNUEABIHF_LINKER=arm-linux-gnueabihf-gcc \
    CARGO_TARGET_RISCV64GC_UNKNOWN_LINUX_GNU_LINKER=riscv64-linux-gnu-gcc \
    PKG_CONFIG_ALLOW_CROSS=1 \
    PKG_CONFIG_SYSROOT_DIR=/ \
    PKG_CONFIG_PATH_x86_64_unknown_linux_gnu=/usr/lib/x86_64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
    PKG_CONFIG_PATH_aarch64_unknown_linux_gnu=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
    PKG_CONFIG_PATH_armv7_unknown_linux_gnueabihf=/usr/lib/arm-linux-gnueabihf/pkgconfig:/usr/share/pkgconfig

ARG ANDROID_CMDLINE_TOOLS_VERSION=15641748
ARG ANDROID_PLATFORM_VERSION=android-37.0
ARG ANDROID_BUILD_TOOLS_VERSION=37.0.0
ARG ANDROID_NDK_VERSION=27.3.13750724
ENV ANDROID_HOME=/opt/android-sdk \
    ANDROID_SDK_ROOT=/opt/android-sdk
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl unzip && \
    mkdir -p "${ANDROID_HOME}/cmdline-tools" && \
    curl -fSL -o /tmp/cmdline-tools.zip \
        "https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_CMDLINE_TOOLS_VERSION}_latest.zip" && \
    unzip -q /tmp/cmdline-tools.zip -d "${ANDROID_HOME}/cmdline-tools" && \
    mv "${ANDROID_HOME}/cmdline-tools/cmdline-tools" "${ANDROID_HOME}/cmdline-tools/latest" && \
    yes | "${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager" --licenses >/dev/null && \
    "${ANDROID_HOME}/cmdline-tools/latest/bin/sdkmanager" \
        "platform-tools" \
        "platforms;${ANDROID_PLATFORM_VERSION}" \
        "build-tools;${ANDROID_BUILD_TOOLS_VERSION}" \
        "ndk;${ANDROID_NDK_VERSION}" >/dev/null && \
    apt-get -y remove --purge curl unzip && apt-get -y autoremove && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* "${ANDROID_HOME}/.android"
ENV ANDROID_NDK_ROOT=/opt/android-sdk/ndk/${ANDROID_NDK_VERSION} \
    ANDROID_NDK_HOME=/opt/android-sdk/ndk/${ANDROID_NDK_VERSION} \
    PATH="/opt/android-sdk/cmdline-tools/latest/bin:/opt/android-sdk/platform-tools:${PATH}"

# renovate: datasource=github-releases depName=WebAssembly/binaryen
ARG BINARYEN_VERSION=version_130
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    ARCH="${TARGETARCH}" && \
    if [ "${TARGETARCH}" = "arm64" ]; then ARCH="aarch64"; fi && \
    if [ "${TARGETARCH}" = "amd64" ]; then ARCH="x86_64"; fi && \
    curl -fSL -o /tmp/binaryen.tar.gz \
        "https://github.com/WebAssembly/binaryen/releases/download/${BINARYEN_VERSION}/binaryen-${BINARYEN_VERSION}-${ARCH}-linux.tar.gz" && \
    tar -xzf /tmp/binaryen.tar.gz -C /opt && \
    mv "/opt/binaryen-${BINARYEN_VERSION}" /opt/binaryen && \
    ln -s /opt/binaryen/bin/wasm-opt /usr/local/bin/wasm-opt && \
    apt-get -y remove --purge curl && apt-get -y autoremove && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
