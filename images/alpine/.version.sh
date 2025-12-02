#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20230901
export VERSION=${ALPINE_VERSION%%-*}