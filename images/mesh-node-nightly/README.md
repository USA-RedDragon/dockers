# Mesh Node (nightly)

The rolling-snapshot build of the [mesh-node](../mesh-node/README.md) image,
running **unmodified [AREDN®](https://www.arednmesh.org/) firmware**. It tracks
AREDN®'s latest x86-64 snapshot (nightly), which is built on a newer OpenWRT
than the tagged releases.

Same approach as [mesh-node](../mesh-node/README.md) - AREDN®'s official x86-64
rootfs is extracted and run **unmodified**, vendoring no AREDN® code - but it
pulls a pinned snapshot rather than a tagged release. The image tag is the
snapshot id (e.g. `20260620-a66a654e`), pinned in the build and bumped by
renovate so the tag matches the rootfs that was built.

## Running

```sh
docker run --rm -it --cap-add NET_ADMIN ghcr.io/usa-reddragon/mesh-node-nightly:latest
```

See [mesh-node](../mesh-node/README.md) for capability requirements and caveats;
they apply identically here.

## Architectures

`linux/amd64` only.

## Trademark

AREDN® is a registered trademark of Amateur Radio Emergency Data Network, Inc.
This project is independent and is **not affiliated with, endorsed by, or
sanctioned by** AREDN, Inc.; the name is used only nominatively, to identify the
upstream firmware that this image packages.

This image redistributes the **official, unmodified** AREDN® x86-64 snapshot
rootfs. It does not modify AREDN® source or builds, and it is **not an official
AREDN® software release**. AREDN, Inc.'s trademark terms reserve the marks for
officially sanctioned releases (see
[AREDNLicense.txt](https://github.com/aredn/aredn/blob/main/AREDNLicense.txt));
should AREDN, Inc. request it, use of the marks here will be discontinued.
