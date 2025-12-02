#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20231219
export VERSION=${ALPINE_VERSION%%-*}