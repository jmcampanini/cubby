.PHONY: build test lint check clean

BIN ?= cubby

build:
	go build -o $(BIN) ./cmd/cubby

test:
	go test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go vet ./...; \
	fi

check: build test lint

clean:
	rm -f $(BIN)
