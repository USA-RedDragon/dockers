#!/bin/bash
alpine_branch=$(sed -nE 's/^FROM alpine:([0-9]+\.[0-9]+).*/v\1/p' "$(dirname "${BASH_SOURCE[0]}")/Dockerfile" | head -1)
GETTEXT_VERSION=$(for repo in main community; do
    curl -sfL "https://dl-cdn.alpinelinux.org/alpine/${alpine_branch}/${repo}/x86_64/APKINDEX.tar.gz" \
        | tar -xzOf - APKINDEX 2>/dev/null \
        | awk '/^P:gettext-envsubst$/ { found = 1 } found && /^V:/ { print substr($0, 3); exit }'
done | head -1)
[ -n "${GETTEXT_VERSION}" ] || { echo "could not resolve gettext-envsubst version for Alpine ${alpine_branch}" >&2; exit 1; }
export GETTEXT_VERSION
export VERSION=${GETTEXT_VERSION%%-*}
