FROM python:3.14.4-alpine3.23@sha256:dd4d2bd5b53d9b25a51da13addf2be586beebd5387e289e798e4083d94ca837a

# renovate: datasource=github-releases depName=markqvist/Reticulum
ARG RNS_VERSION=1.1.9

COPY requirements.txt /tmp/requirements.txt
RUN wget -O /tmp/rns-${RNS_VERSION}-py3-none-any.whl https://github.com/markqvist/Reticulum/releases/download/${RNS_VERSION}/rns-${RNS_VERSION}-py3-none-any.whl \
    && pip install --no-deps --no-cache-dir -r /tmp/requirements.txt \
    && pip install --no-index --no-cache-dir /tmp/rns-${RNS_VERSION}-py3-none-any.whl \
    && rm -f /tmp/rns-${RNS_VERSION}-py3-none-any.whl

EXPOSE 4242
