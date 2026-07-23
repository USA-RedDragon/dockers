FROM jumpmind/symmetricds:3.17.7@sha256:c2922ccd6bdffbbe254b0ee65434abd24ac118c8ebc80c409f70ed9e66365b8f

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
