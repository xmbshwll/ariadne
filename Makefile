.PHONY: help build run validate-spotify-auth validate-apple-music-official validate-tidal-official test test-race lint lint-fix fmt verify deps clean

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_CONFIG ?= .golangci.yml
BINARY ?= ariadne
CMD ?= ./cmd/ariadne
BUILD_DIR ?= bin

help:
	@echo "Available targets:"
	@echo "  build      Build the CLI binary"
	@echo "  run        Run the CLI entrypoint"
	@echo "  validate-spotify-auth  Fetch authenticated Spotify validation artifacts"
	@echo "  validate-apple-music-official  Fetch official Apple Music validation artifacts"
	@echo "  validate-tidal-official  Fetch official TIDAL validation artifacts"
	@echo "  test       Run unit tests"
	@echo "  test-race  Run tests with the race detector"
	@echo "  lint       Run golangci-lint with project config"
	@echo "  lint-fix   Run golangci-lint with autofix enabled"
	@echo "  fmt        Format Go code with gofmt"
	@echo "  verify     Run fmt, lint, and race tests"
	@echo "  deps       Tidy module dependencies"
	@echo "  clean      Remove build artifacts"

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

run:
	$(GO) run $(CMD)

validate-spotify-auth:
	$(GO) run ./cmd/validate-spotify-auth

validate-apple-music-official:
	$(GO) run ./cmd/validate-apple-music-official

validate-tidal-official:
	$(GO) run ./cmd/validate-tidal-official

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

lint:
	$(GOLANGCI_LINT) run --config $(GOLANGCI_LINT_CONFIG) ./...

lint-fix:
	$(GOLANGCI_LINT) run --config $(GOLANGCI_LINT_CONFIG) --fix ./...

fmt:
	gofmt -w .

verify: fmt lint test-race

deps:
	$(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)
