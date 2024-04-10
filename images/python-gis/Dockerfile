FROM python:3.12.3-slim@sha256:5552f03f049ef868437b8db636a6f3f7a74720093fb2f09cfa9232adc78f5945

ENV PYTHONUNBUFFERED=1

# renovate: datasource=repology versioning=deb depName=debian_12/geos
ARG LIBGEOS_VERSION=3.11.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/proj
ARG PROJ_VERSION=9.1.1-1+b1
# renovate: datasource=repology versioning=deb depName=debian_12/openblas
ARG OPENBLAS_VERSION=0.3.21+ds-4

ARG DEV_PACKAGES=" \
    libgeos-dev="${LIBGEOS_VERSION}" \
    libproj-dev="${PROJ_VERSION}" \
    libopenblas-dev="${OPENBLAS_VERSION}" \
"

# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ARG CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/python-pyproj
ARG PYPROJ_VERSION=3.4.1-1+b1

ARG RUNTIME_DEPS=" \
    ca-certificates="${CA_CERTIFICATES_VERSION}" \
    libgeos-c1v5="${LIBGEOS_VERSION}" \
    libproj25="${PROJ_VERSION}" \
    proj-bin="${PROJ_VERSION}" \
    python3-pyproj="${PYPROJ_VERSION}" \
    libopenblas0="${OPENBLAS_VERSION}" \
"

# renovate: datasource=repology versioning=deb depName=debian_12/make-dfsg
ARG MAKE_VERSION=4.3-4.1
# renovate: datasource=repology versioning=deb depName=debian_12/gcc
ARG GCC_VERSION=4:12.2.0-3
# renovate: datasource=repology versioning=deb depName=debian_12/git
ARG GIT_VERSION=1:2.39.2-1.1

ARG BUILD_DEPS=" \
    make=${MAKE_VERSION} \
    g++=${GCC_VERSION} \
    git=${GIT_VERSION} \
    gfortran=${GCC_VERSION} \
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
