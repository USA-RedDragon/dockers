FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6@sha256:a5414d0343f8421dedc023a3805c84d4e0bb030fac392599693b88359b51a631

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
