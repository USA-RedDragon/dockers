FROM rust:1.87.0-alpine AS jwt-cli

# renovate: datasource=github-releases depName=mike-engel/jwt-cli
ARG JWT_CLI_VERSION=6.2.0

WORKDIR /usr/src/jwt-cli
RUN apk add --no-cache musl-dev
RUN cargo install "jwt-cli@${JWT_CLI_VERSION}"

FROM quay.io/minio/aistor/minio:RELEASE.2025-06-20T23-57-47Z AS minio

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS patcher

# renovate: datasource=repology depName=alpine_3_21/ca-certificates
ARG CA_CERTIFICATES_VERSION=20241121-r1
# renovate: datasource=repology depName=alpine_3_21/openssl
ARG OPENSSL_VERSION=3.3.3-r0
# renovate: datasource=repology depName=alpine_3_21/curl
ARG CURL_VERSION=8.12.1-r1
# renovate: datasource=repology depName=alpine_3_21/ripgrep
ARG RIPGREP_VERSION=14.1.1-r0
# renovate: datasource=repology depName=alpine_3_21/sd
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
    jwt encode -A ES384 -P lid=minio -P org=minio -P aid=1 -P did=minio -P cap=9223372036854775807 -P plan=ENTERPRISE-PLUS --sub minio@minio.com --exp=+1000Y --secret @private.pem > /minio.license
__EOF__

FROM minio

COPY --from=patcher /minio.license /
COPY --from=patcher /minio /usr/bin/minio
