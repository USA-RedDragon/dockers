FROM debian:12.5-slim as box-arm64

# renovate: datasource=repology versioning=deb depName=debian_12/python3.11
ENV PYTHON3_VERSION=3.11.2-6
# renovate: datasource=repology versioning=deb depName=debian_12/git
ENV GIT_VERSION=1:2.39.2-1.1
# renovate: datasource=repology versioning=deb depName=debian_12/cmake
ENV CMAKE_VERSION=3.25.1-1
# renovate: datasource=repology versioning=deb depName=debian_12/glibc
ENV LIBC6_VERSION=2.36-9+deb12u4
# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ENV CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/gcc
ENV LIBSTDCPP__6_VERSION=12.2.0-14

# After box86 v0.3.5 releases, swap to git-tags
# renovate: datasource=git-refs versioning=git depName=https://github.com/ptitSeb/box86.git
ENV BOX86_VERSION=master
ENV BOX86_REF=74eb4ed9d57fdd08c9ce13d9d2e6bf40aa177baa
# renovate: datasource=git-tags depName=https://github.com/ptitSeb/box64.git
ENV BOX64_VERSION=v0.2.6

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

# DL3003: We don't want to use WORKDIR here because we won't have these dirs in the file image
# DL3008: I'm not versioning the compilers, repology doesn't quite work for those
# hadolint ignore=DL3003,DL3008
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install --yes --no-install-recommends --no-install-suggests \
        # Unversioned
        build-essential \
        gcc-arm-linux-gnueabihf \
        libc6-dev-armhf-cross \
        # Versioned
        python3.11="${PYTHON3_VERSION}" \
        git="${GIT_VERSION}" \
        cmake="${CMAKE_VERSION}" \
        ca-certificates="${CA_CERTIFICATES_VERSION}" \
    && dpkg --add-architecture armhf \
    && apt-get update \
    && apt-get install --yes --no-install-recommends --no-install-suggests \
        libc6:armhf="${LIBC6_VERSION}" \
        libstdc++6:armhf="${LIBSTDCPP__6_VERSION}" \
    && git clone --single-branch https://github.com/ptitSeb/box86.git; mkdir /box86/build \
    && git clone -b "${BOX64_VERSION}" --single-branch https://github.com/ptitSeb/box64.git; mkdir /box64/build \
    && cd /box86 \
    && git checkout "${BOX86_VERSION}" \
    && cd /box86/build \
    && cmake .. -DARM64=1 -DARM_DYNAREC=ON -DCMAKE_BUILD_TYPE=RelWithDebInfo \
    && make -j"$(nproc)" \
    && make install \
    && cd /box64/build \
    && cmake .. -DARM64=1 -DARM_DYNAREC=ON -DCMAKE_BUILD_TYPE=RelWithDebInfo \
    && make -j"$(nproc)" \
    && make install \
    && apt-get remove --yes --purge \
        python3 \
        git \
        build-essential \
        cmake \
        gcc-arm-linux-gnueabihf \
        ca-certificates \
        libc6-dev-armhf-cross \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

FROM debian:12.5-slim as box-amd64

# We don't install box86 here, as it's not needed for the amd64 architecture
# but the ENV DEBUGGER line means that the box86 binary will always be invoked
# So this script just runs the command it's given
RUN printf '#!/bin/sh\nexec "$@"' > /usr/local/bin/box86 \
    && chmod +x /usr/local/bin/box86

ARG TARGETARCH
# Ignoring the lack of a tag here because the tag is defined in the above FROM lines
# and hadolint isn't aware of those.
# hadolint ignore=DL3006
FROM box-${TARGETARCH}

LABEL maintainer="walentinlamonos@gmail.com"
ARG PUID=1000

ENV USER steam
ENV STEAMCMDDIR "/home/${USER}/steamcmd"

ENV DEBUGGER "/usr/local/bin/box86"

# renovate: datasource=repology versioning=deb depName=debian_12/gcc
ENV LIB32STDCPP__6_VERSION=12.2.0-14
# renovate: datasource=repology versioning=deb depName=debian_12/gcc
ENV LIB32GCC_S1_VERSION=12.2.0-14
# renovate: datasource=repology versioning=deb depName=debian_12/ca-certificates
ENV CA_CERTIFICATES_VERSION=20230311
# renovate: datasource=repology versioning=deb depName=debian_12/nano
ENV NANO_VERSION=7.2-1
# renovate: datasource=repology versioning=deb depName=debian_12/curl
ENV CURL_VERSION=7.88.1-10+deb12u5
# renovate: datasource=repology versioning=deb depName=debian_12/glibc
ENV LOCALES_VERSION=2.36-9+deb12u4

SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]

ARG TARGETARCH
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ca-certificates="${CA_CERTIFICATES_VERSION}" \
        nano="${NANO_VERSION}" \
        curl="${CURL_VERSION}" \
        locales="${LOCALES_VERSION}" \
    && if [ "${TARGETARCH}" = "amd64" ]; then \
        apt-get install -y --no-install-recommends --no-install-suggests \
            lib32stdc++6="${LIB32STDCPP__6_VERSION}" \
            lib32gcc-s1="${LIB32GCC_S1_VERSION}" \
        ; \
    fi \
    && sed -i -e 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen \
    && dpkg-reconfigure locales \
    # Create unprivileged user
    && useradd -u "${PUID}" -m "${USER}" -l \
    && rm -rf /var/lib/apt/lists/*

# Symlink steamclient.so; So misconfigured dedicated servers can find it
RUN ln -s "${STEAMCMDDIR}/linux64/steamclient.so" "/usr/lib/x86_64-linux-gnu/steamclient.so"

USER ${USER}
WORKDIR ${STEAMCMDDIR}

# Download SteamCMD, execute as user
RUN curl -fsSL 'https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz' | tar xvzf - -C "${STEAMCMDDIR}" \
    && ln -s "${STEAMCMDDIR}/linux32/steamclient.so" "${STEAMCMDDIR}/steamservice.so" \
    && "${STEAMCMDDIR}/steamcmd.sh" +quit \
    && mkdir -p "${HOME}/.steam/sdk32" \
    && ln -s "${STEAMCMDDIR}/linux32/steamclient.so" "${HOME}/.steam/sdk32/steamclient.so" \
    && ln -s "${STEAMCMDDIR}/linux32/steamcmd" "${STEAMCMDDIR}/linux32/steam" \
    && mkdir -p "${HOME}/.steam/sdk64" \
    && ln -s "${STEAMCMDDIR}/linux64/steamclient.so" "${HOME}/.steam/sdk64/steamclient.so" \
    && ln -s "${STEAMCMDDIR}/linux64/steamcmd" "${STEAMCMDDIR}/linux64/steam" \
    && ln -s "${STEAMCMDDIR}/steamcmd.sh" "${STEAMCMDDIR}/steam.sh"

ENTRYPOINT ["/home/steam/steamcmd/steamcmd.sh"]
