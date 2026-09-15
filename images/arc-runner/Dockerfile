FROM ghcr.io/actions/actions-runner:2.335.1@sha256:08c30b0a7105f64bddfc485d2487a22aa03932a791402393352fdf674bda2c29

ARG TARGETARCH
ARG DEBIAN_FRONTEND=noninteractive

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        curl \
        build-essential \
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
        zip \
        unzip \
        libudev-dev \
        mesa-vulkan-drivers \
        xvfb \
        libxml2-dev \
        zlib1g-dev \
        libbz2-dev \
        autoconf \
        automake \
        libtool \
    && rm -rf /var/lib/apt/lists/*

RUN install -d -m 0755 /etc/apt/keyrings \
    && curl -fsSL --retry 5 https://packages.mozilla.org/apt/repo-signing-key.gpg \
        -o /etc/apt/keyrings/packages.mozilla.org.asc \
    && echo "deb [signed-by=/etc/apt/keyrings/packages.mozilla.org.asc] https://packages.mozilla.org/apt mozilla main" \
        > /etc/apt/sources.list.d/mozilla.list \
    && printf 'Package: *\nPin: origin packages.mozilla.org\nPin-Priority: 1000\n' \
        > /etc/apt/preferences.d/mozilla \
    && apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests firefox \
    && rm -rf /var/lib/apt/lists/* \
    && firefox --version

RUN if [ "${TARGETARCH}" = "amd64" ]; then \
        set -eux; \
        curl -fsSL --retry 5 https://dl.google.com/linux/linux_signing_key.pub \
            | gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg; \
        echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] https://dl.google.com/linux/chrome/deb/ stable main" \
            > /etc/apt/sources.list.d/google-chrome.list; \
        apt-get update; \
        apt-get install -y --no-install-recommends --no-install-suggests google-chrome-stable; \
        rm -rf /var/lib/apt/lists/*; \
        ver="$(google-chrome --version | grep -oE '[0-9]+(\.[0-9]+){3}')"; \
        curl -fsSL --retry 5 -o /tmp/chromedriver.zip \
            "https://storage.googleapis.com/chrome-for-testing-public/${ver}/linux64/chromedriver-linux64.zip"; \
        unzip -q -j /tmp/chromedriver.zip chromedriver-linux64/chromedriver -d /usr/bin; \
        rm -f /tmp/chromedriver.zip; \
        chromedriver --version; \
    fi

USER runner

# renovate: datasource=github-tags depName=rust-lang/rustup
ARG RUSTUP_VERSION=1.29.1

# renovate: datasource=github-releases depName=rust-lang/rust
ARG RUST_VERSION=1.97.1
ARG RUST_NIGHTLY=nightly-2026-08-15

ENV RUSTUP_HOME=/home/runner/.rustup \
    CARGO_HOME=/home/runner/.cargo \
    PATH=/home/runner/.cargo/bin:/home/runner/.local/bin:${PATH}
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) triple=x86_64-unknown-linux-gnu ;; \
        arm64) triple=aarch64-unknown-linux-gnu ;; \
        *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    url="https://static.rust-lang.org/rustup/archive/${RUSTUP_VERSION}/${triple}/rustup-init"; \
    curl -fsSL --retry 5 -o /tmp/rustup-init "${url}"; \
    echo "$(curl -fsSL --retry 5 "${url}.sha256" | cut -d' ' -f1)  /tmp/rustup-init" | sha256sum -c -; \
    chmod +x /tmp/rustup-init; \
    /tmp/rustup-init -y --no-modify-path \
        --profile minimal \
        --default-toolchain "${RUST_VERSION}" \
        --component clippy,rustfmt,llvm-tools-preview \
        --target wasm32-unknown-unknown; \
    rm -f /tmp/rustup-init; \
    rustup toolchain install "${RUST_NIGHTLY}" --profile minimal \
        --component rust-src \
        --target wasm32-unknown-unknown; \
    rustup --version; \
    cargo --version

# renovate: datasource=crate depName=wasm-pack
ARG WASM_PACK_VERSION=0.15.0
# renovate: datasource=crate versioning=regex:^(?<major>0)\.(?<minor>8)\.(?<patch>\d+)$ depName=cargo-llvm-cov
ARG CARGO_LLVM_COV_VERSION=0.8.7
RUN set -eux; \
    cargo install --locked wasm-pack --version "${WASM_PACK_VERSION}"; \
    cargo install --locked cargo-llvm-cov --version "${CARGO_LLVM_COV_VERSION}"; \
    rm -rf "${CARGO_HOME}/registry" "${CARGO_HOME}/git"; \
    wasm-pack --version; \
    cargo llvm-cov --version
