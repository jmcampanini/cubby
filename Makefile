.PHONY: build test fmt lint tidy check clean

BUILD_DIR ?= build
BIN ?= $(BUILD_DIR)/cubby

build:
	mkdir -p $(dir $(BIN))
	go build -o $(BIN) .

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
