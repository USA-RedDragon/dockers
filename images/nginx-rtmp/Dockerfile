################################
# common dependencies
################################
# renovate: datasource=repology depName=alpine_3_20/musl
ARG MUSL_VERSION=1.2.5-r0
# renovate: datasource=repology depName=alpine_3_20/curl
ARG CURL_VERSION=8.9.0-r0
# renovate: datasource=repology depName=alpine_3_20/gcc
ARG GCC_VERSION=13.2.1_git20240309-r0
# renovate: datasource=repology depName=alpine_3_20/make
ARG MAKE_VERSION=4.4.1-r2
# renovate: datasource=repology depName=alpine_3_20/openssl
ARG OPENSSL_VERSION=3.3.1-r3
# renovate: datasource=repology depName=alpine_3_20/pcre
ARG PCRE_VERSION=8.45-r3
# renovate: datasource=repology depName=alpine_3_20/zlib
ARG ZLIB_VERSION=1.3.1-r1
# renovate: datasource=repology depName=alpine_3_20/ca-certificates
ARG CA_CERTIFICATES_VERSION=20240705-r0

################################
# ffmpeg runtime dependencies
################################
# renovate: datasource=repology depName=alpine_3_20/fdk-aac
ARG FDK_AAC_VERSION=2.0.2-r4
# renovate: datasource=repology depName=alpine_3_20/freetype
ARG FREETYPE_VERSION=2.13.2-r0
# renovate: datasource=repology depName=alpine_3_20/lame
ARG LAME_VERSION=3.100-r5
# renovate: datasource=repology depName=alpine_3_20/libogg
ARG LIBOGG_VERSION=1.3.5-r5
# renovate: datasource=repology depName=alpine_3_20/libass
ARG LIBASS_VERSION=0.17.2-r0
# renovate: datasource=repology depName=alpine_3_20/libvpx
ARG LIBVPX_VERSION=1.14.1-r0
# renovate: datasource=repology depName=alpine_3_20/libvorbis
ARG LIBVORBIS_VERSION=1.3.7-r2
# renovate: datasource=repology depName=alpine_3_20/libwebp
ARG LIBWEBP_VERSION=1.3.2-r0
# renovate: datasource=repology depName=alpine_3_20/libtheora
ARG LIBTHEORA_VERSION=1.1.1-r18
# renovate: datasource=repology depName=alpine_3_20/opus
ARG OPUS_VERSION=1.5.2-r0
# renovate: datasource=repology depName=alpine_3_20/rtmpdump
ARG RTMPDUMP_VERSION=2.4_git20190330-r4
# renovate: datasource=repology depName=alpine_3_20/x264
ARG X264_VERSION=0.164_git20231001-r0
# renovate: datasource=repology depName=alpine_3_20/x265
ARG X265_VERSION=3.6-r0

FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 AS build-nginx

ARG MUSL_VERSION
ARG CURL_VERSION
ARG GCC_VERSION
ARG MAKE_VERSION
ARG OPENSSL_VERSION
ARG PCRE_VERSION
ARG ZLIB_VERSION
ARG CA_CERTIFICATES_VERSION

# renovate: datasource=git-tags extractVersion=^release-(?<version>.*)$ depName=https://github.com/nginx/nginx.git
ARG NGINX_VERSION=1.27.3
# renovate: datasource=git-tags extractVersion=^v(?<version>.*)$ depName=https://github.com/arut/nginx-rtmp-module.git
ARG NGINX_RTMP_VERSION=1.2.2
# renovate: datasource=repology depName=alpine_3_20/linux-headers
ARG LINUX_HEADERS_VERSION=6.6-r0

# Build dependencies.
RUN apk add --no-cache \
  gcc="${GCC_VERSION}" \
  ca-certificates="${CA_CERTIFICATES_VERSION}" \
  curl="${CURL_VERSION}" \
  linux-headers="${LINUX_HEADERS_VERSION}" \
  make="${MAKE_VERSION}" \
  musl-dev="${MUSL_VERSION}" \
  openssl-dev="${OPENSSL_VERSION}" \
  pcre-dev="${PCRE_VERSION}" \
  zlib-dev="${ZLIB_VERSION}"

WORKDIR /tmp

# Get nginx source.
RUN curl -fSsL https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz -o nginx-${NGINX_VERSION}.tar.gz && \
  tar zxf nginx-${NGINX_VERSION}.tar.gz && \
  rm nginx-${NGINX_VERSION}.tar.gz

# Get nginx-rtmp module.
RUN curl -fSsL https://github.com/arut/nginx-rtmp-module/archive/v${NGINX_RTMP_VERSION}.tar.gz -o v${NGINX_RTMP_VERSION}.tar.gz && \
  tar zxf v${NGINX_RTMP_VERSION}.tar.gz && \
  rm v${NGINX_RTMP_VERSION}.tar.gz

# Compile nginx with nginx-rtmp module.
WORKDIR /tmp/nginx-${NGINX_VERSION}
RUN \
  ./configure \
  --prefix=/usr/local/nginx \
  --add-module=/tmp/nginx-rtmp-module-${NGINX_RTMP_VERSION} \
  --conf-path=/etc/nginx/nginx.conf \
  --with-threads \
  --with-file-aio \
  --with-http_ssl_module \
  --with-debug \
  --with-http_stub_status_module \
  --with-cc-opt="-Wimplicit-fallthrough=0" && \
  make -j"$(nproc)" && \
  make DESTDIR=/out install

###############################
# Build the FFmpeg-build image.
FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 AS build-ffmpeg

ARG MUSL_VERSION
ARG CURL_VERSION
ARG GCC_VERSION
ARG MAKE_VERSION
ARG OPENSSL_VERSION

ARG FDK_AAC_VERSION
ARG FREETYPE_VERSION
ARG LAME_VERSION
ARG LIBOGG_VERSION
ARG LIBASS_VERSION
ARG LIBVPX_VERSION
ARG LIBVORBIS_VERSION
ARG LIBWEBP_VERSION
ARG LIBTHEORA_VERSION
ARG OPUS_VERSION
ARG RTMPDUMP_VERSION
ARG X264_VERSION
ARG X265_VERSION

# renovate: datasource=git-tags extractVersion=^n(?<version>.*)$ depName=git://git.ffmpeg.org/ffmpeg.git
ARG FFMPEG_VERSION=7.1
# renovate: datasource=repology depName=alpine_3_20/pkgconf
ARG PKGCONF_VERSION=2.2.0-r0
# renovate: datasource=repology depName=alpine_3_20/yasm
ARG YASM_VERSION=1.3.0-r4

# FFmpeg build dependencies.
RUN apk add --no-cache \
  gcc="${GCC_VERSION}" \
  musl-dev="${MUSL_VERSION}" \
  curl="${CURL_VERSION}" \
  make="${MAKE_VERSION}" \
  fdk-aac-dev="${FDK_AAC_VERSION}" \
  freetype-dev="${FREETYPE_VERSION}" \
  lame-dev="${LAME_VERSION}" \
  libogg-dev="${LIBOGG_VERSION}" \
  libass-dev="${LIBASS_VERSION}" \
  libvpx-dev="${LIBVPX_VERSION}" \
  libvorbis-dev="${LIBVORBIS_VERSION}" \
  libwebp-dev="${LIBWEBP_VERSION}" \
  libtheora-dev="${LIBTHEORA_VERSION}" \
  opus-dev="${OPUS_VERSION}" \
  openssl-dev="${OPENSSL_VERSION}" \
  rtmpdump-dev="${RTMPDUMP_VERSION}" \
  x264-dev="${X264_VERSION}" \
  x265-dev="${X265_VERSION}" \
  pkgconf="${PKGCONF_VERSION}" \
  yasm="${YASM_VERSION}"

WORKDIR /tmp

# Get FFmpeg source.
RUN curl -fSsL https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VERSION}.tar.gz -o ffmpeg-${FFMPEG_VERSION}.tar.gz && \
  tar zxf ffmpeg-${FFMPEG_VERSION}.tar.gz && \
  rm ffmpeg-${FFMPEG_VERSION}.tar.gz

# Compile ffmpeg.
WORKDIR /tmp/ffmpeg-${FFMPEG_VERSION}
RUN \
  ./configure \
  --prefix=/usr/local \
  --enable-version3 \
  --enable-gpl \
  --enable-nonfree \
  --enable-small \
  --enable-libmp3lame \
  --enable-libx264 \
  --enable-libx265 \
  --enable-libvpx \
  --enable-libtheora \
  --enable-libvorbis \
  --enable-libopus \
  --enable-libfdk-aac \
  --enable-libass \
  --enable-libwebp \
  --enable-postproc \
  --enable-libfreetype \
  --enable-openssl \
  --disable-debug \
  --disable-doc \
  --disable-ffplay \
  --extra-libs="-lpthread -lm" && \
  make -j"$(nproc)" && \
  make DESTDIR=/out install && \
  make DESTDIR=/out distclean

# Cleanup.
RUN rm -rf /var/cache/* /tmp/*

##########################
# Build the release image.
FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5

# Set default ports.
ENV HTTP_PORT 80
ENV HTTPS_PORT 443
ENV RTMP_LISTEN_PORT 1935

ARG OPENSSL_VERSION
ARG PCRE_VERSION
ARG ZLIB_VERSION
ARG CA_CERTIFICATES_VERSION
ARG CURL_VERSION

ARG FDK_AAC_VERSION
ARG FREETYPE_VERSION
ARG LAME_VERSION
ARG LIBOGG_VERSION
ARG LIBASS_VERSION
ARG LIBVPX_VERSION
ARG LIBVORBIS_VERSION
ARG LIBWEBP_VERSION
ARG LIBTHEORA_VERSION
ARG OPUS_VERSION
ARG RTMPDUMP_VERSION
ARG X264_VERSION
ARG X265_VERSION

# renovate: datasource=repology depName=alpine_3_20/gettext
ARG GETTEXT_VERSION=0.22.5-r0

RUN apk add --no-cache \
  ca-certificates="${CA_CERTIFICATES_VERSION}" \
  openssl="${OPENSSL_VERSION}" \
  pcre="${PCRE_VERSION}" \
  zlib="${ZLIB_VERSION}" \
  curl="${CURL_VERSION}" \
  fdk-aac="${FDK_AAC_VERSION}" \
  freetype="${FREETYPE_VERSION}" \
  gettext="${GETTEXT_VERSION}" \
  lame="${LAME_VERSION}" \
  libogg="${LIBOGG_VERSION}" \
  libass="${LIBASS_VERSION}" \
  libvpx="${LIBVPX_VERSION}" \
  libvorbis="${LIBVORBIS_VERSION}" \
  libwebp="${LIBWEBP_VERSION}" \
  libtheora="${LIBTHEORA_VERSION}" \
  opus="${OPUS_VERSION}" \
  rtmpdump="${RTMPDUMP_VERSION}" \
  x264-dev="${X264_VERSION}" \
  x265-dev="${X265_VERSION}"

COPY --from=build-nginx /out /
COPY --from=build-ffmpeg /out /

# Add NGINX path, config and static files.
ENV PATH "${PATH}:/usr/local/nginx/sbin"
COPY nginx.conf /etc/nginx/nginx.conf.template
RUN mkdir -p /opt/data && mkdir /www
COPY static /www/static
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 1935
EXPOSE 80

ENTRYPOINT [ "/entrypoint.sh" ]
