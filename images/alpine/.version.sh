#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=3.24.1
export VERSION=${ALPINE_VERSION%%-*}