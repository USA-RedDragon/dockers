FROM jumpmind/symmetricds:3.17.6@sha256:098b8c23c1e7a26bf9f8daccefd0323662283b5242694f3253f514e1f5574fb6

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
