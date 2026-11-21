FROM jumpmind/symmetricds:3.18.4@sha256:b2fa8c0cf8e3acad0dab55e0a3c6b9d5e92ad92b0a9aa9d43071b10b00eb154d

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
