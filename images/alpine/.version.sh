#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=3.23.2
export VERSION=${ALPINE_VERSION%%-*}