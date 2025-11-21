# renovate: datasource=docker depName=docker.io/mattermost/mattermost-enterprise-edition
ARG MATTERMOST_VERSION=10.12.4
FROM docker.io/mattermost/mattermost-enterprise-edition:$MATTERMOST_VERSION AS mattermost

FROM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS patcher

# renovate: datasource=repology depName=alpine_3_21/sd
ARG SD_VERSION=1.0.0-r0
# renovate: datasource=repology depName=alpine_3_21/ripgrep
ARG RIPGREP_VERSION=14.1.1-r0
ARG MATTERMOST_VERSION

RUN apk add --no-cache \
    sd="${SD_VERSION}" \
    ripgrep="${RIPGREP_VERSION}"

COPY --from=mattermost /mattermost/bin/mattermost /
COPY public-key.pem /tmp/public-key.pem

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
  new="$(cat /tmp/public-key.pem)"

  echo Verifying lengths match >&2
  if [ "${#old}" != "${#new}" ]; then
    echo Lengths must match: "${#old}" != "${#new}" >&2
    exit 1
  fi

  echo Verifying binary has current key >&2
  rg --fixed-strings --quiet --multiline -- "$old" mattermost
  echo Patching binary >&2
  sd --fixed-strings --max-replacements=1 -- "$old" "$new" mattermost
EOT

FROM mattermost
COPY --from=patcher /mattermost /mattermost/bin
