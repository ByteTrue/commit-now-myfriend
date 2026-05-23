GO ?= go
BINARY ?= cnm
DIST_DIR ?= dist/go
# Read the version from package.json so cnm --version matches the npm package.
VERSION ?= $(shell node -p "require('./package.json').version" 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X github.com/ByteTrue/commit-now-myfriend/internal/cli.version=$(VERSION)

.PHONY: go-build go-run go-test go-fmt go-vet go-release-snapshot clean

go-build:
	mkdir -p $(DIST_DIR)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(BINARY) ./cmd/cnm

go-run:
	$(GO) run ./cmd/cnm --help

go-test:
	$(GO) test ./...

go-fmt:
	gofmt -w cmd internal

go-vet:
	$(GO) vet ./...

go-release-snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf $(DIST_DIR) dist/release-local
