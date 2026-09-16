FROM python:3.13.12-slim@sha256:f1927c75e81efd1e091dbd64b6c0ecaa5630b38635a3d1c04034ac636e1f94c8

ENV PYTHONUNBUFFERED=1

ARG DEV_PACKAGES=" \
    libgeos-dev \
    libproj-dev \
    libopenblas-dev \
"

ARG RUNTIME_DEPS=" \
    ca-certificates \
    libgeos-c1v5 \
    libproj25 \
    proj-bin \
    python3-pyproj \
    libopenblas0 \
"

ARG BUILD_DEPS=" \
    make \
    g++ \
    git \
    gfortran \
"

COPY requirements.txt /tmp/requirements.txt

# False-positive from hadolint
# hadolint ignore=DL3008
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get -y --no-install-recommends --no-install-suggests install \
      ${DEV_PACKAGES} \
      ${BUILD_DEPS} \
      ${RUNTIME_DEPS} \
    && pip install --no-cache-dir -r /tmp/requirements.txt \
    && apt-get remove --purge -y \
      ${DEV_PACKAGES} \
      ${BUILD_DEPS} \
    && apt-get clean && rm -rf /tmp/setup /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN rm -f /tmp/requirements.txt
