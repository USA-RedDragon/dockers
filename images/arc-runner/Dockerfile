FROM ghcr.io/actions/actions-runner:2.337.0@sha256:e5496277be5d09bc968b3d64911b74e219ac4a3f2edce956a3ecf9271bea1ef4

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        curl \
        build-essential \
        rustup \
        libssl-dev \
        pkg-config \
        cmake \
        libasound2-dev \
        libsqlite3-dev \
        libproj-dev \
        libtiff-dev \
        sqlite3 \
        libclang-dev \
        python3-requests \
    && rm -rf /var/lib/apt/lists/*

USER runner
