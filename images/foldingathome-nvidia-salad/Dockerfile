FROM debian:13.7-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2 AS fakepci

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
