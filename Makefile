VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build install test vet fmt

build:
	go build -ldflags "$(LDFLAGS)" -o bin/sb ./cmd/sb

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/sb

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
