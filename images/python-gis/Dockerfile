FROM python:3.14.8-slim@sha256:a2b82f3c48559aa0a8446d9af49826b6e2b2016f4cd2afabfe6013ec53729170

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
