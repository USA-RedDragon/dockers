FROM jumpmind/symmetricds:3.17.3@sha256:d3d1fa3ed8e7a50aa68a5c010dd30ff418e97fe30271ec9f0c0eda59aa3d9d0e

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
