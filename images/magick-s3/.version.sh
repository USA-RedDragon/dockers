#!/bin/bash
alpine_branch=$(sed -nE 's/^FROM alpine:([0-9]+\.[0-9]+).*/v\1/p' "$(dirname "${BASH_SOURCE[0]}")/Dockerfile" | head -1)
IMAGEMAGICK_VERSION=$(for repo in main community; do
    curl -sfL "https://dl-cdn.alpinelinux.org/alpine/${alpine_branch}/${repo}/x86_64/APKINDEX.tar.gz" \
        | tar -xzOf - APKINDEX 2>/dev/null \
        | awk '/^P:imagemagick$/ { found = 1 } found && /^V:/ { print substr($0, 3); exit }'
done | head -1)
[ -n "${IMAGEMAGICK_VERSION}" ] || { echo "could not resolve imagemagick version for Alpine ${alpine_branch}" >&2; exit 1; }
export IMAGEMAGICK_VERSION
export VERSION=${IMAGEMAGICK_VERSION%%-*}
