FROM python:3.14.8-slim@sha256:c3e521df8b2b498a7a682e7e18676771cb80c6b75b8699af886b2d554ce40151

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
