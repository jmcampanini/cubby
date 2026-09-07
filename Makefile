.DEFAULT_GOAL := help
.PHONY: help build test fmt fmt-check lint lint-fix tidy tidy-check version-check vuln check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || printf 'unknown')
LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  %-16s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build cubby to build/cubby.
	mkdir -p $(dir $(BIN))
	go build -trimpath -buildvcs=false $(LDFLAGS) -o $(BIN) .

test: ## Run all tests uncached with the race detector.
	go test -count=1 -race ./...

fmt: ## Format Go source files.
	go tool golangci-lint fmt

fmt-check: ## Verify formatting without changing files.
	go tool golangci-lint fmt --diff

lint: ## Run golangci-lint.
	go tool golangci-lint run ./...

lint-fix: ## Run golangci-lint with fixes.
	go tool golangci-lint run --fix ./...

tidy: ## Run go mod tidy.
	go mod tidy

tidy-check: ## Check go.mod/go.sum tidiness without modifying files.
	go mod tidy -diff

version-check: build ## Verify the built binary reports the injected version.
	@case "$(VERSION)" in unknown|n/a|"") echo "degenerate version identity: '$(VERSION)'"; exit 1;; esac
	@out="$$($(BIN) --version)" || exit $$?; \
	if [ "$$out" != "cubby version $(VERSION)" ]; then \
		echo "version mismatch: got '$$out', want 'cubby version $(VERSION)'"; \
		exit 1; \
	fi

vuln: ## Check dependencies and reachable code for known vulnerabilities.
	go tool govulncheck ./...

check: fmt-check tidy-check lint test build version-check vuln ## Run the complete local verification contract.

clean: ## Remove build artifacts, coverage files, and test cache.
	rm -rf $(BUILD_DIR) dist coverage.out coverage.txt profile.out cpu.out mem.out
	go clean -testcache
