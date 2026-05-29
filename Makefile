.PHONY: help build test fmt fmt-check lint lint-fix tidy tidy-check check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"
GOFMT_FILES := $(shell git ls-files '*.go')

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-16s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build cubby to build/cubby.
	mkdir -p $(dir $(BIN))
	go build $(LDFLAGS) -o $(BIN) .

test: ## Run go test -race ./...
	go test -race ./...

fmt: ## Format tracked Go files.
	@if [ -n "$(GOFMT_FILES)" ]; then gofmt -w $(GOFMT_FILES); fi

fmt-check: ## Fail if tracked Go files need gofmt.
	@files="$$(gofmt -l $(GOFMT_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "gofmt needed:"; \
		echo "$$files"; \
		echo "Run: make fmt"; \
		exit 1; \
	fi

lint: ## Run golangci-lint.
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with fixes.
	golangci-lint run --fix ./...

tidy: ## Run go mod tidy.
	go mod tidy

tidy-check: ## Check go.mod/go.sum tidiness without modifying files.
	go mod tidy -diff

check: fmt-check tidy-check lint test ## Run fmt-check, tidy-check, lint, and test.

clean: ## Remove build artifacts, coverage files, and test cache.
	rm -rf $(BUILD_DIR) dist coverage.out coverage.txt profile.out cpu.out mem.out
	go clean -testcache
