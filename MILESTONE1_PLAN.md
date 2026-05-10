# Milestone 1 Draft Plan

Milestone 1 should produce the smallest useful vertical slice: a buildable `cubby` Go CLI that can load one host config, load one registered source config, discover declared profiles, and protect the host repo via `cubby gitignore check` and `cubby gitignore sync`.

Milestone 1 does not implement `link`, `unlink`, source file discovery, symlink creation, conflict handling, or profile selection. Those commands are visible only as not-implemented placeholders; their real behavior starts in later milestones.

## Phase 1: Project Skeleton and Tooling

**Goal:** Establish the repository as a normal Go project with repeatable local checks.

**Tasks:**

- Initialize the Go module.
- Add `cmd/cubby/main.go` as the executable entry point.
- Add conventional internal package directories as needed.
- Add a `Makefile` with `build`, `test`, `lint`, and `check` targets.
- Add `golangci-lint` configuration.

**Done when:**

- `make build` produces a runnable `cubby` binary.
- `make test`, `make lint`, and `make check` run successfully, even if tests are still minimal.

## Phase 2: CLI Shell and Command Surface

**Goal:** Expose the v0.1 command hierarchy in help output while only implementing the Milestone 1 behavior.

**Tasks:**

- Add the root `cubby` command.
- Add command groups and commands for the v0.1 surface:
  - `link`
  - `unlink`
  - `prune`
  - `status`
  - `doctor`
  - `profile list`
  - `source list`
  - `gitignore check`
  - `gitignore sync`
  - `lazygit`
- Implement `gitignore check` and `gitignore sync`.
- Leave non-Milestone-1 commands as clear, non-zero "not implemented" commands while keeping them visible in help.
- Standardize basic output and exit-code behavior.

**Done when:**

- `cubby --help` shows the expected v0.1 command surface.
- `cubby gitignore --help`, `cubby gitignore check --help`, and `cubby gitignore sync --help` are useful.
- Non-Milestone-1 commands exit non-zero with a clear `not implemented` message and do not perform partial side effects.

## Phase 3: Minimal Config Loading

**Goal:** Load enough real configuration to drive gitignore checks from one registered source.

**Tasks:**

- Define host config models for `.cubby.toml` with one or more `[[source]]` entries containing `name`, `path`, optional `profiles`, and optional `fail_on_conflict`, while Milestone 1 tests exercise one source.
- Parse host `profiles` and `fail_on_conflict` now so the config shape matches `SPEC.md`, even though Milestone 1 does not yet use them for linking.
- Define source config models for `cubby.toml` with `profiles` and optional `ignore`; parse `ignore` for schema compatibility, though Milestone 1 gitignore behavior uses only declared profiles.
- Load the host config from the current host repo.
- Resolve source paths, including absolute paths, `~` expansion, and relative paths resolved from the discovered host root.
- Load each registered source repo's `cubby.toml`.
- Validate obvious errors with clear messages: missing host config, missing source config, missing source path, and no declared profiles.

**Done when:**

- A host `.cubby.toml` can register one source repo.
- The source repo's `cubby.toml` can declare `profiles = ["work"]`.
- Commands have access to the union of profiles declared by registered sources.
- `gitignore check` and `gitignore sync` compute required patterns from source-declared profiles, not from host `source.profiles` selections.

## Phase 4: Gitignore Guard Engine

**Goal:** Implement the first real end-to-end behavior: detect and repair missing profile ignore patterns.

**Tasks:**

- Compute required patterns for each declared profile:
  - `*.<profile>.*`
  - `*.<profile>`
- Read the host repo's `.gitignore`; treat a missing file as empty.
- Match existing `.gitignore` entries as exact, trimmed, non-comment whole lines; broader glob interpretation is out of scope for Milestone 1.
- Implement `gitignore check` to report missing patterns and exit non-zero when any are absent.
- Implement `gitignore sync` to append missing patterns without duplicating existing patterns.
- Keep `gitignore check/sync` output script-friendly: missing patterns are printed plainly, errors go to stderr, and no machine-readable format is required in Milestone 1.
- Keep writes stable and readable, including newline handling.

**Done when:**

- With no `.gitignore`, `cubby gitignore check` treats it as empty, reports missing `*.work.*` and `*.work` for a source declaring `work`, and exits non-zero.
- `cubby gitignore sync` creates or appends to `.gitignore` and adds both missing patterns with stable, readable newline handling.
- Running `cubby gitignore sync` a second time does not duplicate existing patterns.
- A second `cubby gitignore check` exits successfully.

## Phase 5: Tests and End-to-End Harness

**Goal:** Prove the vertical slice with automated tests that use real filesystem state and real command execution.

**Tasks:**

- Add focused unit tests for profile-union and gitignore pattern logic.
- Add CLI help-output tests verifying `cubby --help` exposes the v0.1 command surface and `cubby gitignore --help` exposes `check` and `sync`.
- Add an end-to-end test that creates temporary host and source repos.
- Seed `src/cubby.toml` with `profiles = ["work"]`.
- Seed `host/.cubby.toml` with one registered source.
- Build the `cubby` binary into a temporary test directory and run that binary with the command working directory set to the temporary host repo.
- Assert `gitignore check` fails before sync and reports both missing patterns.
- Add integration tests for key config errors: missing host `.cubby.toml`, missing source config, missing source path, and source config with no declared profiles.
- Add an end-to-end case or assertion that runs the binary from a nested directory inside the host repo and verifies host root discovery still updates the root `.gitignore`.
- Assert `gitignore sync` creates or appends `.gitignore`, writes both required patterns, and leaves existing content readable.
- Assert a second `gitignore sync` does not duplicate patterns.
- Assert `gitignore check` passes after sync.

**Done when:**

- `make test` exercises the real compiled CLI binary, config loading, filesystem writes, working-directory behavior, and exit codes.
- The Milestone 1 acceptance behavior is covered by at least one end-to-end test.

## Phase 6: Milestone Polish and Verification

**Goal:** Finish the slice by tightening errors, lint, and repeatability.

**Tasks:**

- Run `make check` and fix failures.
- Confirm generated binary behavior manually against a tiny host/source setup.
- Review command output for script-friendly formatting.
- Keep deferred functionality explicitly out of scope rather than half-implemented.

**Done when:**

- `make build`, `make test`, `make lint`, and `make check` all pass.
- `cubby --help` matches the planned v0.1 surface.
- Milestone 1 can be demoed from a clean clone using only the documented commands.

## Implementation Decisions

- Use Cobra for the command hierarchy, nested help output, and command flags.
- Use `github.com/jmcampanini/go-config-loader` from the start, behind an `internal/config` package that owns Cubby's config structs and loading APIs.
- Discover the host root by walking upward from the current directory for `.cubby.toml`.
- Support multiple registered sources internally from the start for profile union/gitignore, while keeping the Milestone 1 acceptance test focused on a single source.
