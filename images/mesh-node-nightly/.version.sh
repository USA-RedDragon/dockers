#!/bin/sh
# The nightly tracks whatever snapshot AREDN currently publishes (the mirror only
# keeps the current one), so the tag is resolved at build time, not pinned here.
VERSION=$(curl -sfL https://downloads.arednmesh.org/snapshots/targets/x86/64/profiles.json \
    | sed -n 's/.*"version_number": *"\([^"]*\)".*/\1/p')
[ -n "${VERSION}" ] || { echo "could not resolve the current AREDN snapshot" >&2; exit 1; }
export VERSION
