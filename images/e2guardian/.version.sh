#!/bin/sh
# renovate: datasource=github-releases depName=e2guardian/e2guardian
export E2GUARDIAN_VERSION=5.5.4r
VERSION="$(echo $E2GUARDIAN_VERSION | sed 's/.$//')"
export VERSION
