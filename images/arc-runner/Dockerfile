FROM ghcr.io/actions/actions-runner:2.335.0@sha256:ccdcb5f6ad3ff02cda20d47544b9bc335b9d555b535fb3556da83e823652a8b4

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
