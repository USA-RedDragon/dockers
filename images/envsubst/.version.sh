#!/bin/bash
# renovate: datasource=repology depName=alpine_3_20/gettext
export GETTEXT_VERSION=0.22.5-r0
export VERSION=${GETTEXT_VERSION%%-*}
