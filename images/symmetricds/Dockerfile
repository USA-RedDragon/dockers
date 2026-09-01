FROM jumpmind/symmetricds:3.17.8@sha256:0c643af0c207f2d12cd777ec22f53a8b6fcb6dbdca03062cfe76db030baad77d

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
