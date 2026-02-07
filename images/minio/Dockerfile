FROM rust:1.91.1-alpine AS jwt-cli

# renovate: datasource=github-releases depName=mike-engel/jwt-cli
ARG JWT_CLI_VERSION=6.2.0

WORKDIR /usr/src/jwt-cli
RUN apk add --no-cache musl-dev
RUN cargo install "jwt-cli@${JWT_CLI_VERSION}"

FROM quay.io/minio/aistor/minio:RELEASE.2026-02-07T07-43-34Z@sha256:33cf8200d428a9b334867c2f873a68a5ceffeb6930753f91bbb1eb188ea1ef43 AS minio

FROM alpine:3.22.2@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS patcher

# renovate: datasource=repology depName=alpine_3_22/ca-certificates
ARG CA_CERTIFICATES_VERSION=20250619-r0
# renovate: datasource=repology depName=alpine_3_22/openssl
ARG OPENSSL_VERSION=3.3.4-r0
# renovate: datasource=repology depName=alpine_3_22/curl
ARG CURL_VERSION=8.12.1-r1
# renovate: datasource=repology depName=alpine_3_22/ripgrep
ARG RIPGREP_VERSION=14.1.1-r0
# renovate: datasource=repology depName=alpine_3_22/sd
ARG SD_VERSION=1.0.0-r0

COPY --from=jwt-cli /usr/local/cargo/bin/jwt /usr/bin/jwt

RUN apk add --no-cache \
    ca-certificates="${CA_CERTIFICATES_VERSION}" \
    openssl="${OPENSSL_VERSION}" \
    curl="${CURL_VERSION}" \
    ripgrep="${RIPGREP_VERSION}" \
    sd="${SD_VERSION}"

COPY --from=minio /usr/bin/minio /

RUN <<__EOF__
    set -eu
    PRIVATE_KEY_PEM="$(openssl ecparam -noout -name secp384r1 -genkey -out -)"
    trap "rm -f private.pem" EXIT ERR
    touch private.pem
    chmod 600 private.pem
    openssl pkcs8 -topk8 -nocrypt -in <(echo -n "${PRIVATE_KEY_PEM}") -out private.pem
    new="$(openssl ec -in <(echo -n "${PRIVATE_KEY_PEM}") -pubout -out -)"
    old="$(curl -fsSL https://subnet.min.io/downloads/license-pubkey.pem)"
    rg --fixed-strings --quiet --multiline -- "$old" /minio
    [ "${#old}" -ne "${#new}" ] && ( echo "old and new strings are not the same length"; exit 1 )
    sd --fixed-strings --max-replacements=1 -- "$old" "$new" /minio
    jwt encode -A ES384 -P lid=minio -P org=minio -P aid=1 -P did=minio -P cap=9223372036854775807 -P plan=ENTERPRISE-PLUS --iss subnet@min.io --exp=+1000Y --secret @private.pem > /minio.license
__EOF__

FROM minio

ENV MINIO_UPDATE=false

COPY --from=patcher /minio.license /
COPY --from=patcher /minio /usr/bin/minio
