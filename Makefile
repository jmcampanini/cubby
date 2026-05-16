.PHONY: build test fmt lint tidy check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/jmcampanini/cubby/cmd.Version=$(VERSION)"

build:
	mkdir -p $(dir $(BIN))
	go build $(LDFLAGS) -o $(BIN) .

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go vet ./...; \
	fi

tidy:
	go mod tidy

check: fmt tidy test lint

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache
