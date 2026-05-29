FROM jumpmind/symmetricds:3.17.5@sha256:bccdcb2907d3e59a7340f542faef7d9eac07417a07ec5adb1c31d68a43a1bdc9

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
