FROM debian:12.15-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc AS fakepci

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY fakepci.c /src/fakepci.c
RUN gcc -O2 -shared -fPIC -o /src/libfakepci.so /src/fakepci.c -ldl

FROM ghcr.io/usa-reddragon/foldingathome-nvidia:v8.5.6 AS base

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl inotify-tools rclone sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Flatten into new layers so CI's gzip setting applies to every layer; SaladCloud cannot pull zstd layers.
FROM scratch

COPY --from=base / /
COPY --from=fakepci /src/libfakepci.so /usr/local/lib/libfakepci.so
COPY --chmod=755 salad-start.sh /usr/local/bin/salad-start
COPY --chmod=755 salad-sync.sh /usr/local/bin/salad-sync
COPY --chmod=755 salad-ws.pl /usr/local/bin/salad-ws

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

USER fah-client
WORKDIR /var/lib/fah-client

ENTRYPOINT ["/usr/local/bin/salad-start"]
CMD []
