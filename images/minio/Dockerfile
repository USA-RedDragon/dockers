FROM quay.io/minio/minio:RELEASE.2024-08-03T04-33-23Z AS minio

# Use ubuntu because sd on debian:latest is too old
FROM ubuntu:24.10 AS patcher

# renovate: datasource=repology versioning=deb depName=ubuntu_24_10/ca-certificates
ARG CA_CERTIFICATES_VERSION=20240203
# renovate: datasource=repology versioning=deb depName=ubuntu_24_10/openssl
ARG OPENSSL_VERSION=3.2.2-1ubuntu3
# renovate: datasource=repology versioning=deb depName=ubuntu_24_10/curl
ARG CURL_VERSION=8.8.0-3ubuntu3
# renovate: datasource=repology versioning=deb depName=ubuntu_24_10/rust-ripgrep
ARG RIPGREP_VERSION=14.1.0-1
# renovate: datasource=repology versioning=deb depName=ubuntu_24_10/rust-sd
ARG SD_VERSION=1.0.0-3

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ca-certificates="${CA_CERTIFICATES_VERSION}" \
        openssl="${OPENSSL_VERSION}" \
        curl="${CURL_VERSION}" \
        ripgrep="${RIPGREP_VERSION}" \
        sd="${SD_VERSION}" \
    && rm -rf /var/lib/apt/lists/*

COPY --from=minio /usr/bin/minio /

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN <<__EOF__
    set -eu
    PRIVATE_KEY_PEM="$(openssl ecparam -noout -name secp384r1 -genkey -out -)"
    trap "rm -f private.pem" EXIT ERR
    touch private.pem
    chmod 600 private.pem
    curl -fSsL https://github.com/mike-engel/jwt-cli/releases/download/6.1.0/jwt-linux.tar.gz -o jwt-linux.tar.gz
    tar -xvzf jwt-linux.tar.gz
    rm -rf jwt-linux.tar.gz
    chmod +x jwt
    PRIVATE_KEY_PKCS8="$(openssl pkcs8 -topk8 -nocrypt -in <(echo -n "${PRIVATE_KEY_PEM}") -out private.pem)"
    new="$(openssl ec -in <(echo -n "${PRIVATE_KEY_PEM}") -pubout -out -)"
    old="$(curl -fsSL https://subnet.min.io/downloads/license-pubkey.pem)"
    rg --fixed-strings --quiet --multiline -- "$old" /minio
    [ "${#old}" -ne "${#new}" ] && ( echo "old and new strings are not the same length"; exit 1 )
    sd --fixed-strings --max-replacements=1 -- "$old" "$new" /minio
    JWT="$(./jwt encode -A ES384 -P lid=minio -P org=minio -P aid=1 -P did=minio -P cap=9223372036854775807 -P plan=ENTERPRISE-PLUS --sub minio@minio.com --exp=+100Y --secret @private.pem)"
    echo -e "#!/bin/bash\nset -euo pipefail\necho \"127.0.0.1 subnet.min.io\" >> /etc/hosts\nexport MINIO_SUBNET_LICENSE=\"${JWT}\"\nexec \"\$@\"" > /start.sh
__EOF__

FROM minio

COPY --from=patcher /start.sh /start.sh
COPY --from=patcher /minio /usr/bin/minio

ENTRYPOINT ["/bin/bash", "/start.sh", "/usr/bin/docker-entrypoint.sh"]
CMD ["minio"]