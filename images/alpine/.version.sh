#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=3.23.3
export VERSION=${ALPINE_VERSION%%-*}