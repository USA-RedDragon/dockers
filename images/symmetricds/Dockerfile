FROM jumpmind/symmetricds:3.17.4@sha256:b94865b822db3f5710267a9742851110be3a52d93d8bb21ca5ad06bca5a12704

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
