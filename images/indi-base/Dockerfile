FROM ubuntu:24.04@sha256:1e622c5f073b4f6bfad6632f2616c7f59ef256e96fe78bf6a595d1dc4376ac02

ARG DEBIAN_FRONTEND=noninteractive

# renovate: datasource=github-releases depName=indilib/indi
ARG INDI_VERSION=v2.1.3
# renovate: datasource=github-releases depName=indilib/indi-3rdparty
ARG INDI_3RD_PARTY_VERSION=v2.1.3

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        build-essential \
        clang \
        git \
        ca-certificates \
        cmake \
        fxload \
        pkg-config \
        ffmpeg \
        libavcodec-dev \
        libavcodec60 \
        libavdevice-dev \
        libavdevice60 \
        libboost-dev \
        libboost-regex-dev \
        libboost-regex1.74.0 \
        libcfitsio-dev \
        libcfitsio10 \
        libcurl4-gnutls-dev \
        libcurl3-gnutls \
        libdc1394-dev \
        libdc1394-25 \
        libev-dev \
        libev4 \
        libfftw3-dev \
        libfftw3-double3 \
        libftdi1-dev \
        libftdi1 \
        libftdi-dev \
        libgmock-dev \
        libgphoto2-dev \
        libgphoto2-6 \
        libgps-dev \
        libgps30 \
        libgsl-dev \
        libgsl27 \
        libjpeg-dev \
        libjpeg8 \
        liblimesuite-dev \
        liblimesuite23.11-1 \
        libnova-dev \
        libnova-0.16-0 \
        libraw-dev \
        libraw23 \
        librtlsdr-dev \
        librtlsdr2 \
        libtheora-dev \
        libtheora0 \
        libtiff-dev \
        libtiff6 \
        libusb-1.0-0-dev \
        libusb-1.0-0 \
        libnutclient-dev \
        libnutclient2 \
        libzmq3-dev \
        libzmq5 \
        zlib1g-dev \
        zlib1g \
    && export CC=clang \
    && export CXX=clang++ \
    && git clone --branch ${INDI_VERSION} --depth 1 https://github.com/indilib/indi.git /tmp/indi \
    && cd /tmp/indi \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release .. \
    && make -j "$(nproc)" \
    && make install \
    && cd /tmp \
    && git clone --branch ${INDI_3RD_PARTY_VERSION} --depth 1 https://github.com/indilib/indi-3rdparty.git /tmp/indi-3rdparty \
    && cd /tmp/indi-3rdparty \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release -DBUILD_LIBS=1 .. \
    && make -j "$(nproc)" \
    && make install \
    && cd .. \
    && rm -rf build \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release .. \
    && make -j "$(nproc)" \
    && make install \
    && apt-get purge -y --auto-remove \
        build-essential \
        clang \
        git \
        ca-certificates \
        cmake \
        pkg-config \
        fxload \
        libavcodec-dev \
        libavdevice-dev \
        libboost-dev \
        libboost-regex-dev \
        libcfitsio-dev \
        libcurl4-gnutls-dev \
        libdc1394-dev \
        libev-dev \
        libfftw3-dev \
        libftdi1-dev \
        libftdi-dev \
        libgmock-dev \
        libgphoto2-dev \
        libgps-dev \
        libgsl-dev \
        libjpeg-dev \
        liblimesuite-dev \
        libnova-dev \
        libraw-dev \
        librtlsdr-dev \
        libtheora-dev \
        libtiff-dev \
        libusb-1.0-0-dev \
        libnutclient-dev \
        libzmq3-dev \
        zlib1g-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
