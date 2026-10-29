FROM python:3.15.2-alpine3.23@sha256:4b777652b7ca0a1fae98a847e83e538605d0fb8de3812f3403076aca74368e74

# renovate: datasource=github-releases depName=markqvist/Reticulum
ARG RNS_VERSION=1.5.5

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
