FROM alpine:3.19.1 as builder

WORKDIR /tmp
RUN mkdir -p /out

COPY patches /patches

# Build and install olsrd
RUN apk add --no-cache \
      git \
      build-base \
      linux-headers \
      bison \
      flex \
      zlib-dev \
      lzo-dev \
      binutils \
      openssl-dev \
      curl

RUN git clone https://github.com/USA-RedDragon/olsrd.git \
    && cd olsrd \
    && git checkout aredn \
    && make prefix=/usr \
    && make prefix=/usr DESTDIR=/out install arprefresh_install txtinfo_install jsoninfo_install dot_draw_install watchdog_install nameservice_install \
    && cd .. \
    && rm -rf olsrd

# Build and install vtun
RUN curl -fSsL https://downloads.sourceforge.net/project/vtun/vtun/3.0.3/vtun-3.0.3.tar.gz -o vtun-3.0.3.tar.gz \
    && tar -xzf vtun-3.0.3.tar.gz \
    && rm vtun-3.0.3.tar.gz \
    && cd vtun-3.0.3 \
    # --build=unknown-unknown-linux is magic for cross-compiling
    && ./configure --prefix=/usr --build=unknown-unknown-linux \
    && for patch in /patches/vtun/*.patch; do patch -p1 < $patch; done \
    && make \
    && make DESTDIR=/out install \
    && cd .. \
    && rm -rf vtun-3.0.3

FROM alpine:3.19.1

COPY --from=builder /out /

RUN apk add --no-cache \
    bash \
    curl \
    zlib \
    lzo \
    openssl \
    iptables \
    iproute2 \
    rsyslog \
    logrotate \
    dnsmasq \
    jq \
    gettext \
    wireguard-tools \
    s6 \
    cronie

RUN sed -i 's/module(load="imklog")//g' /etc/rsyslog.conf

RUN curl -fSsL https://raw.githubusercontent.com/aredn/aredn_packages/3.22.12.0/blockknownencryption/files/20-blockknownencryption -o /usr/bin/blockknownencryption \
    && chmod +x /usr/bin/blockknownencryption

COPY --chown=root:root rootfs /

RUN rm -f /etc/periodic/daily/logrotate \
    && (crontab -l ; echo "* * * * * /usr/sbin/logrotate /etc/logrotate.conf") | crontab - \
    && touch /var/log/cron.log
