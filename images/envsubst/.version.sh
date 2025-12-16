#!/bin/bash
# renovate: datasource=repology depName=alpine_3_22/gettext
export GETTEXT_VERSION=0.24.1-r0
export VERSION=${GETTEXT_VERSION%%-*}
