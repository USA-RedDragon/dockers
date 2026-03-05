FROM jumpmind/symmetricds:3.17.0@sha256:dd72271a22fa2a6c4ea59480b4b6e33b635411a9a7e6f843b2f6e3e56c2e9e6b

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
