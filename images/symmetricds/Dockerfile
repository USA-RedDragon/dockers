FROM jumpmind/symmetricds:3.16.9@sha256:e54ed2eaa8a2f51844ff80c479663122b83d6b18b9fc74e7dfc9f607b166ccc7

RUN apk add --no-cache tini

WORKDIR /opt/symmetric-ds
RUN chown -R nobody:nogroup /opt/symmetric-ds

USER nobody:nogroup

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/bin/sh", "-c", "/opt/symmetric-ds/bin/sym_service start & tail -F /opt/symmetric-ds/logs/symmetric.log"]
