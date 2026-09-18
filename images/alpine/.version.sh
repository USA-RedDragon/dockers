#!/bin/bash
# renovate: datasource=docker depName=alpine
export ALPINE_VERSION=3.24.2
export VERSION=${ALPINE_VERSION%%-*}