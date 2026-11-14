FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS builder

RUN apk add --no-cache \
    iperf3

EXPOSE 5201

ENTRYPOINT [ "iperf3", "-s" ]
