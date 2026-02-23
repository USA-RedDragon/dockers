#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=3.21.6
export VERSION=${ALPINE_VERSION%%-*}