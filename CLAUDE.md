# Agent Notes

Use the Makefile targets for routine validation:

- `make help` — list available Makefile targets.
- `make build` — build the `cubby` binary to `build/cubby`. Override with `BUILD_DIR=...` or `BIN=...`.
- `make test` — run all Go tests with `go test -race ./...`.
- `make fmt` — format tracked/non-ignored Go files with `gofmt -w`.
- `make fmt-check` — check gofmt drift without modifying files.
- `make tidy` — run `go mod tidy`.
- `make tidy-check` — check module tidiness with `go mod tidy -diff` without modifying files.
- `make lint` — run `golangci-lint run ./...`.
- `make lint-fix` — run `golangci-lint run --fix ./...`.
- `make check` — run `fmt-check`, `tidy-check`, `lint`, and `test`; use before pushing.
- `make clean` — remove build artifacts, coverage files, and the Go test cache.
