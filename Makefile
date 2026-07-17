# mdit — developer tasks. Run `make` (or `make help`) for the list.
# Recipes assume Go 1.26+. Dev tools install into $(GOBIN) via `make tools`.

BINARY       := mdit
CMD          := ./cmd/mdit
PKG          := ./...
BIN_DIR      := bin
COVERPROFILE := coverage.out

# Version stamped into the binary: git tag/sha, or "dev" outside a repo.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Match CI: static binary, no cgo.
export CGO_ENABLED := 0

# Tool versions. Using @latest for portability; pin to exact tags once you've
# confirmed them (keeps local == CI reproducible). See docs/DEVELOPMENT.md.
GOLANGCI_VERSION := latest
LEFTHOOK_VERSION := latest
GOVULN_VERSION   := latest

GO      ?= go
GOBIN   := $(shell $(GO) env GOPATH)/bin

.DEFAULT_GOAL := help

## ---- Build & run -----------------------------------------------------------

.PHONY: build
build: ## Build the binary into bin/
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) $(CMD)

.PHONY: install
install: ## Install mdit into $(GOBIN)
	$(GO) install -ldflags '$(LDFLAGS)' $(CMD)

.PHONY: run
run: ## Run mdit locally (pass args via ARGS=, e.g. make run ARGS=notes/)
	$(GO) run $(CMD) $(ARGS)

## ---- Test ------------------------------------------------------------------

.PHONY: test
test: ## Run the test suite
	$(GO) test $(PKG)

.PHONY: test-race
test-race: ## Run tests with the race detector (needs cgo)
	CGO_ENABLED=1 $(GO) test -race $(PKG)

.PHONY: cover
cover: ## Run tests and print total coverage
	$(GO) test -covermode=atomic -coverprofile=$(COVERPROFILE) $(PKG)
	$(GO) tool cover -func=$(COVERPROFILE) | tail -1

.PHONY: cover-html
cover-html: cover ## Open the HTML coverage report
	$(GO) tool cover -html=$(COVERPROFILE)

.PHONY: bench
bench: ## Run benchmarks
	$(GO) test -run '^$$' -bench . -benchmem $(PKG)

## ---- Quality ---------------------------------------------------------------

.PHONY: fmt
fmt: ## Format all Go code
	$(GO) fmt $(PKG)

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-clean
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	$(GO) vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: tidy
tidy: ## Tidy and verify go.mod / go.sum
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: vuln
vuln: ## Scan for known vulnerabilities
	govulncheck $(PKG)

## ---- Aggregates ------------------------------------------------------------

.PHONY: check
check: fmt-check vet lint test ## Everything CI enforces (fast local gate)

.PHONY: ci
ci: fmt-check vet lint test-race cover vuln ## Full pipeline mirror

## ---- Setup -----------------------------------------------------------------

.PHONY: tools
tools: ## Install dev tools (golangci-lint, lefthook, govulncheck)
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	$(GO) install github.com/evilmartians/lefthook@$(LEFTHOOK_VERSION)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULN_VERSION)

.PHONY: hooks
hooks: ## Install git hooks via lefthook
	lefthook install

.PHONY: setup
setup: tools hooks ## One-shot dev environment setup
	@echo "dev environment ready — try 'make check'"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(COVERPROFILE)

## ---- Help ------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} \
		/^## / {sub(/^## /, ""); printf "\n\033[1m%s\033[0m\n", $$0; next} \
		/^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' \
		$(MAKEFILE_LIST)
