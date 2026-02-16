FROM jumpmind/symmetricds:3.16.10@sha256:1bbbb450ec83812aabbbe010ff1d19456e83cc18ccc68a4b4c31e340c4435eee

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
