FROM python:3.14.8-slim@sha256:89fb7d3da20043c370643435258bdd7ab755d326d359001d02988ed15ae5219e

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
