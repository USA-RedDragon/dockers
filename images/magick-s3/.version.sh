#!/bin/bash
# renovate: datasource=repology depName=alpine_3_21/imagemagick
export IMAGEMAGICK_VERSION=7.1.1.41-r0
export VERSION=${IMAGEMAGICK_VERSION%%-*}
