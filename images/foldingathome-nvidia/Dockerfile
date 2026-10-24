FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ocl-icd-libopencl1 \
        clinfo \
    && mkdir -p /etc/OpenCL/vendors \
    && printf '%s\n' 'libnvidia-opencl.so.1' \
        > /etc/OpenCL/vendors/nvidia.icd \
    && rm -rf /var/lib/apt/lists/*

USER fah-client
