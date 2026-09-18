FROM alpine:3.24.2@sha256:31b6477333eb8257db9e5d7c3a7264fd0467928756f0bbcc27d35bea5d28cdbd AS build-nginx

# renovate: datasource=git-tags extractVersion=^release-(?<version>.*)$ depName=https://github.com/nginx/nginx.git
ARG NGINX_VERSION=1.31.6
# renovate: datasource=git-tags extractVersion=^v(?<version>.*)$ depName=https://github.com/arut/nginx-rtmp-module.git
ARG NGINX_RTMP_VERSION=1.2.2

# Build dependencies.
RUN apk add --no-cache \
  gcc \
  ca-certificates \
  curl \
  linux-headers \
  make \
  musl-dev \
  openssl-dev \
  pcre-dev \
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
  make DESTDIR=/out install

###############################
# Build the FFmpeg-build image.
FROM alpine:3.24.2@sha256:31b6477333eb8257db9e5d7c3a7264fd0467928756f0bbcc27d35bea5d28cdbd AS build-ffmpeg

# renovate: datasource=git-tags extractVersion=^n(?<version>.*)$ depName=git://git.ffmpeg.org/ffmpeg.git
ARG FFMPEG_VERSION=7.1.5

# FFmpeg build dependencies.
RUN apk add --no-cache \
  gcc \
  musl-dev \
  curl \
  make \
  fdk-aac-dev \
  freetype-dev \
  lame-dev \
  libogg-dev \
  libass-dev \
  libvpx-dev \
  libvorbis-dev \
  libwebp-dev \
  libtheora-dev \
  opus-dev \
  openssl-dev \
  rtmpdump-dev \
  x264-dev \
  x265-dev \
  pkgconf \
  yasm

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
FROM alpine:3.24.2@sha256:31b6477333eb8257db9e5d7c3a7264fd0467928756f0bbcc27d35bea5d28cdbd

# Set default ports.
ENV HTTP_PORT=80
ENV HTTPS_PORT=443
ENV RTMP_LISTEN_PORT=1935

RUN apk add --no-cache \
  ca-certificates \
  openssl \
  pcre \
  zlib \
  curl \
  fdk-aac \
  freetype \
  gettext \
  lame \
  libogg \
  libass \
  libvpx \
  libvorbis \
  libwebp \
  libtheora \
  opus \
  rtmpdump \
  x264-dev \
  x265-dev

COPY --from=build-nginx /out /
COPY --from=build-ffmpeg /out /

# Add NGINX path, config and static files.
ENV PATH="${PATH}:/usr/local/nginx/sbin"
COPY nginx.conf /etc/nginx/nginx.conf.template
RUN mkdir -p /opt/data && mkdir /www
COPY static /www/static
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 1935
EXPOSE 80

ENTRYPOINT [ "/entrypoint.sh" ]
