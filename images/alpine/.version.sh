#!/bin/bash
# renovate: datasource=docker depName=docker.io/alpine
export ALPINE_VERSION=20230329
export VERSION=${ALPINE_VERSION%%-*}