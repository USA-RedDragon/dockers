FROM ghcr.io/actions/actions-runner:2.336.0@sha256:0cfdcc701ce933c6d243c6b0b2da767366dc9f2e99961d4c3754b0b78084cdda

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
