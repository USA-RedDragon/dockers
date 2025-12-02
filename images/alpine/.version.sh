#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20240315
export VERSION=${ALPINE_VERSION%%-*}