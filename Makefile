.PHONY: help build test fmt fmt-check lint lint-fix tidy tidy-check check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"
GO_FILES := $(shell git ls-files '*.go' && git ls-files --others --exclude-standard '*.go')

help:
	@printf '%s\n' \
		'Available targets:' \
		'  build       Build cubby to $(BIN)' \
		'  test        Run go test -race ./...' \
		'  lint        Run golangci-lint' \
		'  lint-fix    Run golangci-lint with fixes' \
		'  fmt         Format tracked/non-ignored Go files' \
		'  fmt-check   Check gofmt without modifying files' \
		'  tidy        Run go mod tidy' \
		'  tidy-check  Check go.mod/go.sum tidiness without modifying files' \
		'  check       Run fmt-check, tidy-check, lint, and test' \
		'  clean       Remove build artifacts, coverage files, and test cache'

build:
	mkdir -p $(dir $(BIN))
	go build $(LDFLAGS) -o $(BIN) .

test:
	go test -race ./...

fmt:
	@if [ -n "$(GO_FILES)" ]; then gofmt -w $(GO_FILES); fi

fmt-check:
	@if [ -n "$(GO_FILES)" ]; then \
		unformatted="$$(gofmt -l $(GO_FILES))"; \
		if [ -n "$$unformatted" ]; then \
			printf 'gofmt needed:\n%s\n' "$$unformatted"; \
			exit 1; \
		fi; \
	fi

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

tidy:
	go mod tidy

tidy-check:
	go mod tidy -diff

check: fmt-check tidy-check lint test

clean:
	rm -rf $(BUILD_DIR) dist coverage.out coverage.txt profile.out cpu.out mem.out
	go clean -testcache
