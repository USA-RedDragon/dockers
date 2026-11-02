FROM debian:12.15-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS fakepci

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY fakepci.c /src/fakepci.c
RUN gcc -O2 -shared -fPIC -o /src/libfakepci.so /src/fakepci.c -ldl

FROM ghcr.io/usa-reddragon/foldingathome-nvidia:v8.5.6 AS base

# Flatten into new layers so CI's gzip setting applies to every layer; SaladCloud cannot pull zstd layers.
FROM scratch

COPY --from=base / /
COPY --from=fakepci /src/libfakepci.so /usr/local/lib/libfakepci.so
COPY --chmod=755 salad-start.sh /usr/local/bin/salad-start

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

USER fah-client
WORKDIR /var/lib/fah-client

ENTRYPOINT ["/usr/local/bin/salad-start"]
CMD []
