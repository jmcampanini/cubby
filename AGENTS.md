# Agent Notes

Use the Makefile targets for routine validation:

- `make help` - list all available Makefile targets.

Some other targets are:

- `make build` - build the `cubby` binary to `build/cubby`. Override with `BUILD_DIR=...` or `BIN=...`.
- `make check` - run `fmt-check`, `tidy-check`, `lint`, and `test`; use before pushing.
- `make clean` - remove build artifacts, coverage files, and the Go test cache.
