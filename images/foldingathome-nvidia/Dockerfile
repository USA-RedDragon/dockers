FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6@sha256:87ea7a44c13bc61fa45cd16e443bb3ae30587ef49abad55df2d937cfe7fad879

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ocl-icd-libopencl1 \
        clinfo \
    && mkdir -p /etc/OpenCL/vendors \
    && printf '%s\n' 'libnvidia-opencl.so.1' \
        > /etc/OpenCL/vendors/nvidia.icd \
    && ln -sf libOpenCL.so.1 /usr/lib/x86_64-linux-gnu/libOpenCL.so \
    && rm -rf /var/lib/apt/lists/*

USER fah-client
