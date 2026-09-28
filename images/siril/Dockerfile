FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS builder

# renovate: datasource=gitlab-tags depName=free-astro/siril
ARG SIRIL_VERSION=1.4.4

RUN apk add --no-cache \
        build-base \
        cairo-dev \
        cfitsio-dev \
        cmake \
        curl-dev \
        fftw-dev \
        git \
        glib-dev \
        gsl-dev \
        gtk+3.0-dev \
        gtksourceview4-dev \
        lcms2-dev \
        libjpeg-turbo-dev \
        libpng-dev \
        libxisf-dev \
        meson \
        opencv-dev \
        pax-utils \
        samurai \
        tiff-dev \
        wcslib-dev \
        yyjson-dev

WORKDIR /src
RUN git clone --depth 1 --branch "${SIRIL_VERSION}" https://gitlab.com/free-astro/siril.git . \
    && git submodule update --init --depth 1 subprojects/librtprocess subprojects/yyjson

# Headless processing only needs FITS, XISF, the common image formats and
# libcurl (online star catalogues for plate solving). Video, camera raw,
# HEIF/JXL, EXIF and the git script repository are left out.
RUN meson setup _build \
        --prefix=/usr \
        --buildtype=release \
        -Dopenmp=true \
        -DlibXISF=true \
        -Dlibcurl=true \
        -Dlibpng=true \
        -Dlibjpeg=true \
        -Dlibtiff=true \
        -Dexiv2=false \
        -Dlibgit2=false \
        -Dlibraw=false \
        -Dlibjxl=false \
        -Dlibheif=false \
        -Dffms2=false \
        -Dffmpeg=false \
    && meson compile -C _build \
    && DESTDIR=/out meson install -C _build \
    && rm -rf /out/usr/share/applications /out/usr/share/icons /out/usr/share/metainfo /out/usr/share/man

# Record the shared libraries the installed binaries need that Siril doesn't
# ship itself, as apk "so:" dependencies for the runtime stage.
RUN scanelf --needed --nobanner --format '%n#p' --recursive /out/usr \
        | tr ',' '\n' \
        | sort -u \
        | while read -r lib; do \
            [ -e "/out/usr/lib/${lib}" ] || echo "so:${lib}"; \
          done > /out/runtime-deps \
    && cat /out/runtime-deps

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

COPY --from=builder /out/runtime-deps /tmp/runtime-deps
RUN apk add --no-cache ca-certificates $(cat /tmp/runtime-deps) \
    && rm /tmp/runtime-deps
COPY --from=builder /out/usr /usr

# Siril keeps its settings under HOME; point it somewhere writable so the
# image works with a read-only root filesystem and /tmp mounted.
ENV HOME=/tmp XDG_CONFIG_HOME=/tmp/.config XDG_CACHE_HOME=/tmp/.cache

RUN siril-cli --version

CMD ["siril-cli"]
