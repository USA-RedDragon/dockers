FROM ghcr.io/usa-reddragon/foldingathome:v8.5.6@sha256:fcbb675bf711f94577590d1f1232285c1d4e985676214d678b62acbca4ffabaf

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
