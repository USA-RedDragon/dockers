FROM debian:12.15-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8 AS fakepci

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY fakepci.c /src/fakepci.c
RUN gcc -O2 -shared -fPIC -o /src/libfakepci.so /src/fakepci.c -ldl

FROM ghcr.io/usa-reddragon/foldingathome-nvidia:v8.5.6

COPY --from=fakepci /src/libfakepci.so /usr/local/lib/libfakepci.so
COPY --chmod=755 salad-start.sh /usr/local/bin/salad-start

ENTRYPOINT ["/usr/local/bin/salad-start"]
CMD []
