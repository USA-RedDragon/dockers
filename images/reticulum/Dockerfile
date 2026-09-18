FROM python:3.14.7-alpine3.23@sha256:200baf0bf6b7904cf643358326e580ca0b8afec82416fd6d233746ceb215283c

# renovate: datasource=github-releases depName=markqvist/Reticulum
ARG RNS_VERSION=1.5.2

RUN apk add --no-cache git

USER nobody:nogroup

ENV HOME=/app
ENV PATH="${HOME}/.local/bin:${PATH}"
WORKDIR /app

COPY requirements.txt /tmp/requirements.txt
RUN wget -O /tmp/rns-${RNS_VERSION}-py3-none-any.whl https://github.com/markqvist/Reticulum/releases/download/${RNS_VERSION}/rns-${RNS_VERSION}-py3-none-any.whl \
    && pip install --user --no-deps --no-cache-dir -r /tmp/requirements.txt \
    && pip install --user --no-index --no-cache-dir /tmp/rns-${RNS_VERSION}-py3-none-any.whl \
    && rm -f /tmp/rns-${RNS_VERSION}-py3-none-any.whl

EXPOSE 4242

ENTRYPOINT [ "rnsd" ]
