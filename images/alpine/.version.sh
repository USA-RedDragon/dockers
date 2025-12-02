#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20201218
export VERSION=${ALPINE_VERSION%%-*}