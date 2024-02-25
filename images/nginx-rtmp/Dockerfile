# renovate: datasource=git-tags extractVersion=^release-(?<version>.*)$ depName=https://github.com/nginx/nginx.git
ARG NGINX_VERSION=1.25.4
# renovate: datasource=git-tags extractVersion=^v(?<version>.*)$ depName=https://github.com/arut/nginx-rtmp-module.git
ARG NGINX_RTMP_VERSION=1.2.2
# renovate: datasource=git-tags extractVersion=^n(?<version>.*)$ depName=git://git.ffmpeg.org/ffmpeg.git
ARG FFMPEG_VERSION=6.1.1

##############################
# Build the NGINX-build image.
FROM alpine:3.19.1@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0 as build-nginx
ARG NGINX_VERSION
ARG NGINX_RTMP_VERSION

# Build dependencies.
RUN apk add --no-cache \
  build-base \
  ca-certificates \
  curl \
  gcc \
  libc-dev \
  libgcc \
  linux-headers \
  make \
  musl-dev \
  openssl \
  openssl-dev \
  pcre \
  pcre-dev \
  pkgconf \
  pkgconfig \
  zlib-dev

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
  make install

###############################
# Build the FFmpeg-build image.
FROM alpine:3.19.1@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0 as build-ffmpeg
ARG FFMPEG_VERSION
ARG PREFIX=/usr/local

# FFmpeg build dependencies.
RUN apk add --no-cache \
  build-base \
  coreutils \
  curl \
  freetype-dev \
  lame-dev \
  libogg-dev \
  libass \
  libass-dev \
  libvpx-dev \
  libvorbis-dev \
  libwebp-dev \
  libtheora-dev \
  openssl-dev \
  opus-dev \
  pkgconf \
  pkgconfig \
  rtmpdump-dev \
  x264-dev \
  x265-dev \
  yasm

RUN echo https://dl-cdn.alpinelinux.org/alpine/edge/community >> /etc/apk/repositories
RUN apk add --no-cache fdk-aac-dev

WORKDIR /tmp

# Get FFmpeg source.
RUN curl -fSsL https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VERSION}.tar.gz -o ffmpeg-${FFMPEG_VERSION}.tar.gz && \
  tar zxf ffmpeg-${FFMPEG_VERSION}.tar.gz && \
  rm ffmpeg-${FFMPEG_VERSION}.tar.gz

# Compile ffmpeg.
WORKDIR /tmp/ffmpeg-${FFMPEG_VERSION}
RUN \
  ./configure \
  --prefix=${PREFIX} \
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
  make install && \
  make distclean

# Cleanup.
RUN rm -rf /var/cache/* /tmp/*

##########################
# Build the release image.
FROM alpine:3.19.1@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0

# Set default ports.
ENV HTTP_PORT 80
ENV HTTPS_PORT 443
ENV RTMP_LISTEN_PORT 1935

RUN apk add --no-cache \
  ca-certificates \
  gettext \
  openssl \
  pcre \
  lame \
  libogg \
  curl \
  libass \
  libvpx \
  libvorbis \
  libwebp \
  libtheora \
  opus \
  rtmpdump \
  x264-dev \
  x265-dev

COPY --from=build-nginx /usr/local/nginx /usr/local/nginx
COPY --from=build-nginx /etc/nginx /etc/nginx
COPY --from=build-ffmpeg /usr/local /usr/local
COPY --from=build-ffmpeg /usr/lib/libfdk-aac.so.2 /usr/lib/libfdk-aac.so.2

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
