FROM ghcr.io/mend/renovate-ee-server:10.0.0

USER root

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN <<__EOF__
    set -eu
    NAME="Renovate"
    EMAIL="renovate@mend.io"
    COMPANY="Mend.io"
    DATE="$(TZ='UTC' date -d '+1000 year' +'%FT%TZ')"
    BASE64_PAYLOAD="$(echo -n \{\"name\":\""${NAME}"\",\"email\":\""${EMAIL}"\",\"company\":\""${COMPANY}"\",\"expiry\":\""${DATE}"\",\"enterprise\": true\} | base64 -w0)"
    PRIVATE_KEY_BASE64="$(openssl ecparam -name prime256v1 -genkey -noout -out - | base64 -w0)"
    PUBLIC_KEY_BASE64="$(openssl ec -in <(echo -n "${PRIVATE_KEY_BASE64}" | base64 -d) -pubout -out - | base64 -w0)"
    SIG_BASE64="$(openssl dgst -sha1 -sign <(echo -n "${PRIVATE_KEY_BASE64}" | base64 -d) <(echo -n "${BASE64_PAYLOAD}") | base64 -w0)"
    openssl dgst -sha1 -verify <(echo -n "${PUBLIC_KEY_BASE64}" | base64 -d) -signature <(echo -n "${SIG_BASE64}" | base64 -d) <(echo -n "${BASE64_PAYLOAD}")
    SIGNATURE="$(echo -n "${SIG_BASE64}" | base64 -d | od -An -v -tx1 | tr -d ' \n')"
    PUBLIC_KEY_HEX="$(openssl ec -conv_form uncompressed -in <(echo -n "${PUBLIC_KEY_BASE64}" | base64 -d) -pubin -outform der | tail -c 65 | od -An -v -tx1 | tr -d ' \n')"
    sed -E -i "s/\{pub:\"04[a-fA-F0-9]{128}\"/\{pub:\"""${PUBLIC_KEY_HEX}""\"/" /usr/src/mend/src/enterprise-server.js
    echo -e "#!/bin/bash\nset -euo pipefail\nexport MEND_RNV_LICENSE_KEY=\"$BASE64_PAYLOAD.$SIGNATURE\"\nexec \"\$@\"" > /start.sh
__EOF__

USER ubuntu

ENTRYPOINT ["/bin/bash", "/start.sh", "docker-entrypoint.sh"]
CMD ["node", "src/enterprise-server.js"]
