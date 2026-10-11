FROM ghcr.io/bdd/runitor:v1.4.1-alpine@sha256:2b8314456908065f5d79cd25c7c310ddecd52995cea9d2a3526a4b0b19b4dc94 AS runitor

FROM ghcr.io/gabe565/moreutils:0.6.1@sha256:ae74fb352762709efdb79a67d1b51ab78d6fd04b2e39eb6d53e218674e7dc4e5 AS moreutils

FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS go-deps
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY internal internal

FROM go-deps AS wrapper
ARG TARGETOS
ARG TARGETARCH
COPY cmd/restic-wrapper cmd/restic-wrapper
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags='-w -s' -trimpath ./cmd/restic-wrapper

FROM go-deps AS dumpdb
ARG TARGETOS
ARG TARGETARCH
COPY cmd/dumpdb cmd/dumpdb
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags='-w -s' -trimpath ./cmd/dumpdb

FROM go-deps AS restoredb
ARG TARGETOS
ARG TARGETARCH
COPY cmd/restoredb cmd/restoredb
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags='-w -s' -trimpath ./cmd/restoredb

FROM restic/restic:0.19.1@sha256:136600b6ff6843d61d355f7f71f460a166429f35de6fd11b568fece3c9a4d510
RUN test -f /usr/bin/restic

RUN apk add --no-cache bash postgresql-client mariadb-client mongodb-tools sqlite

COPY --from=runitor /usr/local/bin/runitor /usr/bin/runitor
COPY --from=moreutils /usr/bin/ts /usr/bin/ts
COPY --from=wrapper /app/restic-wrapper /usr/local/bin/restic
COPY --from=dumpdb /app/dumpdb /usr/bin/dumpdb
COPY --from=restoredb /app/restoredb /usr/bin/restoredb
COPY rootfs /

ENV KUBECONFIG=/.kube/config
ENV RESTIC_GROUP_BY=tags
ENV RESTIC_CACHE_DIR=/cache
