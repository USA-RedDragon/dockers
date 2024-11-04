FROM debian:12.6-slim@sha256:2ccc7e39b0a6f504d252f807da1fc4b5bcd838e83e4dec3e2f57b2a4a64e7214 AS build-3rdparty

ARG DEBIAN_FRONTEND=noninteractive

# renovate: datasource=github-releases depName=indilib/indi
ARG INDI_VERSION=v2.1.0
# renovate: datasource=github-releases depName=indilib/indi-3rdparty
ARG INDI_3RD_PARTY_VERSION=v2.1.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        build-essential \
        git \
        ca-certificates \
        cmake \
        fxload \
        pkg-config \
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
    && make DESTDIR=/indi-drivers install

FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5

# renovate: datasource=github-releases depName=indilib/indi
ARG INDI_VERSION=v2.1.0
# renovate: datasource=github-releases depName=indilib/indi-3rdparty
ARG INDI_3RD_PARTY_VERSION=v2.1.0

WORKDIR /tmp

RUN apk add --no-cache --virtual .build-deps \
        build-base \
        linux-headers \
        cmake \
        git \
        zlib-dev \
        libev-dev \
        libnova-dev \
        libusb-dev \
        curl-dev \
        gsl-dev \
        libjpeg-turbo-dev \
        fftw-dev \
        cfitsio-dev \
        ffmpeg-dev \
        libftdi1-dev \
        libraw-dev \
        libdc1394-dev \
        libgphoto2-dev \
        gpsd-dev \
        zeromq-dev \
    && apk add --no-cache \
        zlib \
        libev \
        libnova \
        libusb \
        curl \
        gsl \
        libjpeg-turbo \
        fftw \
        cfitsio \
        ffmpeg \
        libftdi1 \
        libraw \
        libdc1394 \
        libgphoto2 \
        gpsd \
        zeromq \
        gcompat \
    && git clone --branch ${INDI_VERSION} --depth 1 https://github.com/indilib/indi.git /tmp/indi \
    && cd /tmp/indi \
    && sed -i '/^#define HAS_EVENT_FD/a #include <sys\/select\.h>' libs/sockets/select.h \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release .. \
    && make -j "$(nproc)" \
    && make install \
    && cd /tmp \
    && git clone --branch ${INDI_3RD_PARTY_VERSION} --depth 1 https://github.com/indilib/indi-3rdparty.git /tmp/indi-3rdparty \
    && cd /tmp/indi-3rdparty \
    && sed -i '/^#pragma once/a #include <stdint\.h>' libqsi/Filter.h \
    && mkdir build \
    && cd build \
    && cmake -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release -DBUILD_LIBS=1 .. \
    && make -j "$(nproc)" \
    && make install \
    && apk del .build-deps \
    && cd / \
    && rm -rf /tmp/*

WORKDIR /

COPY --from=build-3rdparty /indi-drivers /
