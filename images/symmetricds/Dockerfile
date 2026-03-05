FROM jumpmind/symmetricds:3.16.11@sha256:26a4b9f62e71fd6d43cc8b1d79b1335ce08eda62104e84519afa7ce0ed0d4c96

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
