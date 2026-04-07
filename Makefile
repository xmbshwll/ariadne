.PHONY: help build run validate-spotify-auth validate-apple-music-official validate-tidal-official test test-race test-release lint lint-fix fmt verify verify-release deps clean

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_CONFIG ?= .golangci.yml
BINARY ?= ariadne
CMD_MODULE_DIR ?= cmd
CLI_PACKAGE ?= ./ariadne
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
	@echo "  test-release  Run tests with GOWORK=off to verify each module independently"
	@echo "  lint       Run golangci-lint with project config"
	@echo "  lint-fix   Run golangci-lint with autofix enabled"
	@echo "  fmt        Format Go code with gofmt"
	@echo "  verify     Run fmt, lint, and race tests"
	@echo "  verify-release  Run release-oriented module verification"
	@echo "  deps       Tidy module dependencies"
	@echo "  clean      Remove build artifacts"

build:
	@mkdir -p $(BUILD_DIR)
	cd $(CMD_MODULE_DIR) && $(GO) build -o ../$(BUILD_DIR)/$(BINARY) $(CLI_PACKAGE)

run:
	cd $(CMD_MODULE_DIR) && $(GO) run $(CLI_PACKAGE)

validate-spotify-auth:
	cd $(CMD_MODULE_DIR) && $(GO) run ./validate-spotify-auth

validate-apple-music-official:
	cd $(CMD_MODULE_DIR) && $(GO) run ./validate-apple-music-official

validate-tidal-official:
	cd $(CMD_MODULE_DIR) && $(GO) run ./validate-tidal-official

test:
	$(GO) test ./...
	cd $(CMD_MODULE_DIR) && $(GO) test ./...

test-race:
	$(GO) test -race ./...
	cd $(CMD_MODULE_DIR) && $(GO) test -race ./...

test-release:
	GOWORK=off $(GO) test ./...
	cd $(CMD_MODULE_DIR) && GOWORK=off $(GO) test ./...

lint:
	$(GOLANGCI_LINT) run --config $(GOLANGCI_LINT_CONFIG) ./...
	cd $(CMD_MODULE_DIR) && $(GOLANGCI_LINT) run --config ../$(GOLANGCI_LINT_CONFIG) ./...

lint-fix:
	$(GOLANGCI_LINT) run --config $(GOLANGCI_LINT_CONFIG) --fix ./...
	cd $(CMD_MODULE_DIR) && $(GOLANGCI_LINT) run --config ../$(GOLANGCI_LINT_CONFIG) --fix ./...

fmt:
	gofmt -w .

verify: fmt lint test-race

verify-release: test-release
	cd $(CMD_MODULE_DIR) && GOWORK=off $(GO) build ./...

deps:
	$(GO) mod tidy
	cd $(CMD_MODULE_DIR) && $(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)
