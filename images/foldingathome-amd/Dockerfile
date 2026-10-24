FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6

USER root

ARG ROCM_APT_VERSION=7.1.1

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        gnupg \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://repo.radeon.com/rocm/rocm.gpg.key \
        | gpg --dearmor -o /etc/apt/keyrings/rocm.gpg \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/${ROCM_APT_VERSION}/ jammy main" \
        > /etc/apt/sources.list.d/rocm.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        rocm-opencl-runtime \
    && rm -rf /var/lib/apt/lists/*

ENV LD_LIBRARY_PATH=/opt/rocm/lib

USER fah-client
