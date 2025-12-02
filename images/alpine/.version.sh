#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20210730
export VERSION=${ALPINE_VERSION%%-*}