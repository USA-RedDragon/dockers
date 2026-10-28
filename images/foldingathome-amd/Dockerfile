FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6@sha256:fcbb675bf711f94577590d1f1232285c1d4e985676214d678b62acbca4ffabaf

USER root

ARG ROCM_APT_VERSION=6.4.4

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl \
        gnupg \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://repo.radeon.com/rocm/rocm.gpg.key \
        | gpg --dearmor -o /etc/apt/keyrings/rocm.gpg \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/${ROCM_APT_VERSION}/ jammy main" \
        > /etc/apt/sources.list.d/rocm.list \
    && printf 'Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600\n' \
        > /etc/apt/preferences.d/rocm-pin-600 \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        hip-runtime-amd \
        rocm-opencl-runtime \
    && ln -sf libOpenCL.so.1 /usr/lib/x86_64-linux-gnu/libOpenCL.so \
    && rm -rf /var/lib/apt/lists/*

ENV LD_LIBRARY_PATH=/opt/rocm/lib

USER fah-client
