FROM archlinux:base-devel@sha256:992c7760458aabf9e0318cfdf78f8213afbd4d4c1e65ac47b78447a93c614246

RUN pacman-key --init && \
  pacman-key --populate archlinux

RUN cat <<EOF >> /etc/pacman.conf
[usa-reddragon]
SigLevel = Required TrustedOnly
Server = https://raw.githubusercontent.com/USA-RedDragon/arch-packages/bins/x86_64

[multilib]
Include = /etc/pacman.d/mirrorlist
EOF

RUN curl -fSsLo /tmp/usa-reddragon-keyring-20230501-6-any.pkg.tar.zst https://github.com/USA-RedDragon/arch-packages/raw/bins/x86_64/usa-reddragon-keyring-20230501-6-any.pkg.tar.zst && \
  pacman --noconfirm -U /tmp/usa-reddragon-keyring-20230501-6-any.pkg.tar.zst && \
  rm -f /tmp/usa-reddragon-keyring-20230501-6-any.pkg.tar.zst

RUN pacman --noconfirm -Syyu --needed reflector
RUN reflector -p https --delay 0.5 --fastest 15 --sort rate --save /etc/pacman.d/mirrorlist

RUN pacman --noconfirm -Syyu --needed git devtools dbus sudo usa-reddragon-keyring reflector
RUN useradd -m user || true
RUN mkdir -p /home/user/.gnupg
RUN chown -R user:user /home/user
RUN dbus-uuidgen --ensure=/etc/machine-id
RUN sed -i "s|MAKEFLAGS=.*|MAKEFLAGS=-j$(nproc)|" /etc/makepkg.conf

# Add the user to the sudoers list
RUN echo "user ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
