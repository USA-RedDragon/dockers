# AREDN Virtual Node Base Image

This project is a part of the [AREDN Cloud Tunnel](https://github.com/USA-RedDragon/aredn-cloud-tunnel) project. Unless you're developing for the project, you probably want to go there instead.

This project contains DNSMasq, VTun, OLSR and a few packages to make a minimal AREDN-compatible environment. It is designed to be used as a base image for [aredn-manager](https://github.com/USA-RedDragon/aredn-manager) and the [AREDN Cloud Tunnel](https://github.com/USA-RedDragon/aredn-cloud-tunnel).

## Updating Patches

This is just a small note to myself documenting the command to update the patches.

`git format-patch --output-directory ../patches/olsrd v0.9.8..HEAD`

Patches sourced from: <https://github.com/aredn/aredn_packages/tree/develop/net/olsrd/patches> and <https://github.com/aredn/aredn_packages/tree/develop/net/vtun/patches>
