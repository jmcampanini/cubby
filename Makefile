.PHONY: help build test fmt fmt-check lint lint-fix tidy tidy-check check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"
GO_FILES := $(shell git ls-files '*.go' && git ls-files --others --exclude-standard '*.go')

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-16s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build cubby to build/cubby.
	mkdir -p $(dir $(BIN))
	go build $(LDFLAGS) -o $(BIN) .

test: ## Run go test -race ./...
	go test -race ./...

fmt: ## Format tracked/non-ignored Go files.
	@if [ -n "$(GO_FILES)" ]; then gofmt -w $(GO_FILES); fi

fmt-check: ## Check gofmt without modifying files.
	@if [ -n "$(GO_FILES)" ]; then \
		unformatted="$$(gofmt -l $(GO_FILES))"; \
		if [ -n "$$unformatted" ]; then \
			printf 'gofmt needed:\n%s\n' "$$unformatted"; \
			exit 1; \
		fi; \
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
