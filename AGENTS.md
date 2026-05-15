# Agent Notes

Use the Makefile targets for routine validation:

- `make build` — build the `cubby` binary to `build/cubby`. Override with `BUILD_DIR=...` or `BIN=...`.
- `make test` — run all Go tests with `go test ./...`.
- `make fmt` — format Go files with `gofmt -w .`.
- `make tidy` — run `go mod tidy`.
- `make lint` — run `golangci-lint` if installed, otherwise `go vet ./...`.
- `make check` — run `fmt`, `tidy`, `test`, and `lint`; use before pushing.
- `make clean` — remove the build directory and clear the Go test cache.
