FROM ghcr.io/actions/actions-runner:2.337.0@sha256:e5496277be5d09bc968b3d64911b74e219ac4a3f2edce956a3ecf9271bea1ef4

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
        libsqlite3-dev \
        libproj-dev \
        libtiff-dev \
        sqlite3 \
        libclang-dev \
        python3-requests \
        zip \
        unzip \
        cpio \
        rsync \
        zstd \
        xxd \
        libarchive-tools \
    && rm -rf /var/lib/apt/lists/*

# renovate: datasource=node-version depName=node
ARG NODE_VERSION=24.21.0
# renovate: datasource=github-tags depName=aws/aws-cli
ARG AWSCLI_VERSION=2.36.49
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) node_arch=x64; aws_arch=x86_64 ;; \
        arm64) node_arch=arm64; aws_arch=aarch64 ;; \
        *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    tarball="node-v${NODE_VERSION}-linux-${node_arch}.tar.xz"; \
    curl -fsSL --retry 5 -o "/tmp/${tarball}" "https://nodejs.org/dist/v${NODE_VERSION}/${tarball}"; \
    curl -fsSL --retry 5 "https://nodejs.org/dist/v${NODE_VERSION}/SHASUMS256.txt" \
        | grep " ${tarball}\$" | sed "s| .*|  /tmp/${tarball}|" | sha256sum -c -; \
    tar -xJf "/tmp/${tarball}" -C /usr/local --strip-components=1 --no-same-owner \
        --exclude=CHANGELOG.md --exclude=LICENSE --exclude=README.md; \
    rm -f "/tmp/${tarball}"; \
    node --version; npm --version; \
    curl -fsSL --retry 5 -o /tmp/awscli.zip \
        "https://awscli.amazonaws.com/awscli-exe-linux-${aws_arch}-${AWSCLI_VERSION}.zip"; \
    unzip -q /tmp/awscli.zip -d /tmp/awscli; \
    /tmp/awscli/aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli; \
    rm -rf /tmp/awscli /tmp/awscli.zip; \
    aws --version; \
    install -d -m 0755 /etc/apt/keyrings; \
    curl -fsSL --retry 5 https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        -o /etc/apt/keyrings/githubcli-archive-keyring.gpg; \
    echo "deb [arch=${TARGETARCH} signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
        > /etc/apt/sources.list.d/github-cli.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends --no-install-suggests gh; \
    rm -rf /var/lib/apt/lists/*; \
    gh --version

USER runner

ENV PATH=/home/runner/.local/bin:${PATH}
