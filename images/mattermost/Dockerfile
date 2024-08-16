# renovate: datasource=docker depName=docker.io/mattermost/mattermost-enterprise-edition
ARG MATTERMOST_VERSION=9.11.0
FROM mattermost/mattermost-enterprise-edition:$MATTERMOST_VERSION AS mattermost

FROM alpine:3.20.2@sha256:0a4eaa0eecf5f8c050e5bba433f58c052be7587ee8af3e8b3910ef9ab5fbe9f5 AS patcher

# renovate: datasource=repology depName=alpine_3_20/sd
ARG SD_VERSION=1.0.0-r0
# renovate: datasource=repology depName=alpine_3_20/ripgrep
ARG RIPGREP_VERSION=14.1.0-r0
# renovate: datasource=repology depName=alpine_3_20/openssl
ARG OPENSSL_VERSION=3.3.1-r3
# renovate: datasource=repology depName=alpine_3_20/bash
ARG BASH_VERSION=5.2.26-r0
ARG MATTERMOST_VERSION

RUN apk add --no-cache \
    sd="${SD_VERSION}" \
    ripgrep="${RIPGREP_VERSION}" \
    openssl="${OPENSSL_VERSION}" \
    bash="${BASH_VERSION}"

COPY --from=mattermost /mattermost/bin/mattermost /

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

RUN <<EOT
  set -eu

  if [ "$MATTERMOST_VERSION" = latest ]; then
    ref=master
  else
    ref="v$MATTERMOST_VERSION"
  fi

  url="https://github.com/mattermost/mattermost/raw/$ref/server/channels/utils/license-public-key.txt"
  echo Downloading current public key from "$url" >&2
  old="$(wget -q -O- "$url")"

  PRIVKEY="$(openssl genrsa -out - 2048)"
  PUBKEY="$(openssl rsa -in <(echo "$PRIVKEY") -pubout)"
  TODAY="$(TZ='UTC' date +'%s%3N')"
  ISSUED_DATE="$(($TODAY-31556952000))" # - 1 year
  EXPIRATION_DATE="$(($TODAY+31556952000000))" # + 1000 years

  # pragma: allowlist nextline secret
  PAYLOAD="{\"id\":\"ABCDEFGHIJKLMNOPQRSTUVWXYZ\",\"issued_at\":${ISSUED_DATE},\"starts_at\":${ISSUED_DATE},\"expires_at\":${EXPIRATION_DATE},\"customer\":{\"id\":\"ABCDEFGHIJKLMNOPQRSTUVWXYZ\",\"name\":\"\",\"email\":\"\",\"company\":\"\"},\"features\":{\"users\":9223372036854776000,\"ldap\":true,\"ldap_groups\":true,\"mfa\":true,\"google_oauth\":true,\"office365_oauth\":true,\"openid\":true,\"compliance\":true,\"cluster\":true,\"metrics\":true,\"mhpns\":true,\"saml\":true,\"elastic_search\":true,\"announcement\":true,\"theme_management\":true,\"email_notification_contents\":true,\"data_retention\":true,\"message_export\":true,\"custom_permissions_schemes\":true,\"custom_terms_of_service\":true,\"guest_accounts\":true,\"guest_accounts_permissions\":true,\"id_loaded\":true,\"lock_teammate_name_display\":true,\"enterprise_plugins\":true,\"advanced_logging\":true,\"cloud\":false,\"shared_channels\":true,\"remote_cluster_service\":true,\"outgoing_oauth_connections\":true,\"future_features\":true},\"sku_name\":\"enterprise\",\"sku_short_name\":\"enterprise\",\"is_trial\":false,\"is_gov_sku\":false,\"signup_jwt\":null}"
  BASE64_PAYLOAD="$(echo -n "${PAYLOAD}" | base64 -w0)"
  SIGNATURE="$(openssl dgst -sha512 -sign <(echo "${PRIVKEY}") <(echo -n "${BASE64_PAYLOAD}") | base64 -w0)"

  echo Verifying key lengths match >&2
  if [ "${#old}" != "${#PUBKEY}" ]; then
    echo Lengths must match: "${#old}" != "${#PUBKEY}" >&2
    exit 1
  fi

  echo Verifying binary has current key >&2
  rg --fixed-strings --quiet --multiline -- "$old" mattermost
  echo Patching binary >&2
  sd --fixed-strings --max-replacements=1 -- "$old" "$PUBKEY" mattermost

  echo -e "#!/bin/sh\nset -eu\nexport MM_LICENSE=\"${BASE64_PAYLOAD}${SIGNATURE}\"\nexec \"\$@\"" > /start.sh
EOT

FROM mattermost
COPY --from=patcher /mattermost /mattermost/bin
COPY --from=patcher /start.sh /start.sh

ENTRYPOINT ["/bin/sh", "/start.sh", "/entrypoint.sh"]
CMD ["mattermost"]
