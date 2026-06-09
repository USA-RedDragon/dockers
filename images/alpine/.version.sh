#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=3.24.0
export VERSION=${ALPINE_VERSION%%-*}