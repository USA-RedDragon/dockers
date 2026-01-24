FROM atlassian/bitbucket:10.1.4@sha256:07337770f8bf21063699cf4cf4c362ed8247bd893b12542cbdebd8b37e61058c AS bitbucket

FROM alpine:3.22.2@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS patcher

WORKDIR /patcher

# renovate: datasource=repology depName=alpine_3_22/bash
ARG BASH_VERSION=5.2.37-r0
# renovate: datasource=repology depName=alpine_3_22/openssl
ARG OPENSSL_VERSION=3.5.4-r0
# renovate: datasource=repology depName=alpine_3_22/sed
ARG SED_VERSION=4.9-r2
# renovate: datasource=repology depName=alpine_3_22/zip
ARG ZIP_VERSION=3.0-r13

RUN apk add --no-cache \
    bash="${BASH_VERSION}" \
    openssl="${OPENSSL_VERSION}" \
    sed="${SED_VERSION}" \
    zip="${ZIP_VERSION}"

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

COPY --from=bitbucket /opt/atlassian/bitbucket/app/WEB-INF/lib/atlassian-extras-key-manager-*.jar .

RUN <<__DOCKER_EOF__
    openssl dsaparam -out dsaparam.pem 1024
    openssl gendsa -out outer_private_key.pem dsaparam.pem
    openssl gendsa -out inner_private_key.pem dsaparam.pem
    openssl dsa -in outer_private_key.pem -pubout -outform DER | base64 -w0 > license_string_key_b64.txt
    openssl dsa -in inner_private_key.pem -pubout -outform DER | base64 -w0 > license_hash_key_b64.txt

    to_sed_hex() {
        local str="$1"
        local len=${#str}
        # Print length as 4-digit hex (2 bytes), then string as hex, then escape with \x
        (printf "%04x" "$len"; echo -n "$str" | od -A n -t x1 | tr -d ' \n') | sed 's/../\\x&/g'
    }

    OLD_KEY="MIIBuDCCASwGByqGSM44BAEwggEfAoGBAP1/U4EddRIpUt9KnC7s5Of2EbdSPO9EAMMeP4C2USZpRV1AIlH7WT2NWPq/xfW6MPbLm1Vs14E7gB00b/JmYLdrmVClpJ+f6AR7ECLCT7up1/63xhv4O1fnxqimFQ8E+4P208UewwI1VBNaFpEy9nXzrith1yrv8iIDGZ3RSAHHAhUAl2BQjxUjC8yykrmCouuEC/BYHPUCgYEA9+GghdabPd7LvKtcNrhXuXmUr7v6OuqC+VdMCz0HgmdRWVeOutRZT+ZxBxCBgLRJFnEj6EwoFhO3zwkyjMim4TwWeotUfI0o4KOuHiuzpnWRbqN/C/ohNWLx+2J6ASQ7zKTxvqhRkImog9/hWuWfBpKLZl6Ae1UlZAFMO/7PSSoDgYUAAoGBAIvfweZvmGo5otwawI3no7Udanxal3hX2haw962KL/nHQrnC4FG2PvUFf34OecSK1KtHDPQoSQ+DHrfdf6vKUJphw0Kn3gXm4LS8VK/LrY7on/wh2iUobS2XlhuIqEc5mLAUu9Hd+1qxsQkQ50d0lzKrnDqPsM0WA9htkdJJw2nS"  # pragma: allowlist secret
    OLD_HASH="MIIBtzCCASwGByqGSM44BAEwggEfAoGBAP1/U4EddRIpUt9KnC7s5Of2EbdSPO9EAMMeP4C2USZpRV1AIlH7WT2NWPq/xfW6MPbLm1Vs14E7gB00b/JmYLdrmVClpJ+f6AR7ECLCT7up1/63xhv4O1fnxqimFQ8E+4P208UewwI1VBNaFpEy9nXzrith1yrv8iIDGZ3RSAHHAhUAl2BQjxUjC8yykrmCouuEC/BYHPUCgYEA9+GghdabPd7LvKtcNrhXuXmUr7v6OuqC+VdMCz0HgmdRWVeOutRZT+ZxBxCBgLRJFnEj6EwoFhO3zwkyjMim4TwWeotUfI0o4KOuHiuzpnWRbqN/C/ohNWLx+2J6ASQ7zKTxvqhRkImog9/hWuWfBpKLZl6Ae1UlZAFMO/7PSSoDgYQAAoGALZHuJwQzgGnYm/X9BkMcewYQnWjMIGWHd9Yom5Qw7cVIdiZkqpiSzSKurO/WAHHLN31obg7NgGkitWUysECRE3zuJVbKGhx9xjVMnP6z5SwI89vB7Gn7UWxoCvT0JZgcMyQobXeVBtM9J3EgzkdDx/+Dck7uz/l1y+HDNdRzW00="  # pragma: allowlist secret

    NEW_KEY=$(cat license_string_key_b64.txt)
    NEW_HASH=$(cat license_hash_key_b64.txt)

    OLD_KEY_HEX=$(to_sed_hex "$OLD_KEY")
    NEW_KEY_HEX=$(to_sed_hex "$NEW_KEY")
    
    OLD_HASH_HEX=$(to_sed_hex "$OLD_HASH")
    NEW_HASH_HEX=$(to_sed_hex "$NEW_HASH")

    JAR_FILE=$(ls atlassian-extras-key-manager-*.jar)
    
    mkdir -p jar_tmp
    (cd jar_tmp && unzip -q "../$JAR_FILE")
    
    find jar_tmp -name "*.class" -type f -exec sed -i "s/$OLD_KEY_HEX/$NEW_KEY_HEX/g" {} +
    find jar_tmp -name "*.class" -type f -exec sed -i "s/$OLD_HASH_HEX/$NEW_HASH_HEX/g" {} +
    
    (cd jar_tmp && zip -q -r ../atlassian-extras-key-manager-patched.jar .)
    
    mv atlassian-extras-key-manager-patched.jar "$JAR_FILE"
    rm -rf jar_tmp
__DOCKER_EOF__

FROM bitbucket

# renovate: datasource=repology depName=ubuntu_24_04/python-cryptography
ARG CRYPTOGRAPHY_VERSION=41.0.7-4ubuntu0.1
# renovate: datasource=repology depName=ubuntu_24_04/javaproperties
ARG JAVAPROPERTIES_VERSION=0.8.1-1.1

RUN apt-get update && \
    apt-get install -y \
        --no-install-recommends \
        --no-install-suggests \
            python3-cryptography=${CRYPTOGRAPHY_VERSION} \
            python3-javaproperties=${JAVAPROPERTIES_VERSION} && \
    rm -rf /var/lib/apt/lists/*

COPY --from=patcher /patcher/atlassian-extras-key-manager-*.jar /opt/atlassian/bitbucket/app/WEB-INF/lib/
COPY --from=patcher /patcher/outer_private_key.pem /license_keys/outer_private_key.pem
COPY --from=patcher /patcher/inner_private_key.pem /license_keys/inner_private_key.pem
COPY license.py entrypoint.sh /

CMD [ ]
ENTRYPOINT [ "/entrypoint.sh" ]
