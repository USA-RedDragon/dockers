MARIADB_IMAGE ?= mariadb:11
POSTGRES_IMAGE ?= postgres:18-alpine

.PHONY: test
test:
	go vet ./...
	go test ./...

.PHONY: test-mariadb
test-mariadb:
	MARIADB_IMAGE=$(MARIADB_IMAGE) go test -tags mariadb ./cmd/restoredb/mariadb/ -count=1 -v

.PHONY: test-postgres
test-postgres:
	POSTGRES_IMAGE=$(POSTGRES_IMAGE) go test -tags postgres ./cmd/restoredb/postgres/ -count=1 -v
