FROM jumpmind/symmetricds:3.16.7@sha256:9cf1dbdf1929380acd957bdfcd2f4a71491576e9b604297cd12f2f52835c6eca

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
