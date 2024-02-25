FROM python:3.12.2

ENV PYTHONUNBUFFERED=1

COPY requirements.txt /tmp/requirements.txt

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get -y --no-install-recommends install \
      ca-certificates \
      build-essential \
      git \
      libproj-dev \
      libproj25 \
      proj-bin \
      libgeos-c1v5 \
      libgeos-dev \
      libopenblas-dev \
      libopenblas0 \
      python3-pyproj \
      gfortran && \
    pip install -r /tmp/requirements.txt && \
    apt-get remove -y \
      build-essential \
      libgeos-dev \
      libopenblas-dev \
      git \
      gfortran && \
    apt-get clean && rm -rf /tmp/setup /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN rm -f /tmp/requirements.txt
