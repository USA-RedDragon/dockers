FROM jumpmind/symmetricds:3.17.0@sha256:b8057c4bc359756a7042ff78f222bd68fef0b3fa14115918a3ee4761afb71848

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
