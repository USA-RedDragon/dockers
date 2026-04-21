FROM ghcr.io/actions/actions-runner:2.334.0@sha256:b6614fce332517f74d0a76e7c762fb08e4f2ff13dcf333183397c8a5725b6e8e

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
