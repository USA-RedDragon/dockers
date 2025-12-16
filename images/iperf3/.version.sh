#!/bin/bash
# renovate: datasource=repology depName=alpine_3_22/iperf3
export IPERF3_VERSION=3.19.1-r0
export VERSION=${IPERF3_VERSION%%-*}
