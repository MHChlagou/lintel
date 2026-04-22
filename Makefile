GO         ?= go
VERSION    ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE     := github.com/aegis-sec/aegis
LDFLAGS    := -s -w \
  -X $(MODULE)/internal/version.Version=$(VERSION) \
  -X $(MODULE)/internal/version.Commit=$(COMMIT) \
  -X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build test test-race vet fmt fmt-check lint tidy ci clean install smoke

build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/aegis ./cmd/aegis

install: build
	install -m 0755 bin/aegis $$HOME/.aegis/bin/aegis

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

fmt-check:
	@diff=$$(gofmt -l .); \
	if [ -n "$$diff" ]; then echo "unformatted:"; echo "$$diff"; exit 1; fi

lint:
	golangci-lint run --timeout=5m

tidy:
	$(GO) mod tidy

ci: vet fmt-check test-race build

smoke: build
	@bin=$$(pwd)/bin/aegis; \
	tmp=$$(mktemp -d); \
	(cd $$tmp && git init -q && "$$bin" --repo "$$tmp" init); \
	test -f $$tmp/.aegis/aegis.yaml && echo "smoke ok"

clean:
	rm -rf bin dist coverage.out
