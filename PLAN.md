# cubby Implementation Plan

This plan turns `SPEC.md` into vertical, testable slices. Each milestone should
produce a knowably working behavior that crosses the real CLI, config loading,
filesystem state, and command exit codes. Shared internals can grow as needed,
but the plan favors end-to-end proof over building hidden layers first.

## Milestone 1: Tooling, CLI Shell, and First End-to-End Guard

**Outcome:** `cubby` is a buildable Go CLI with project checks in place and one
real end-to-end behavior: protecting the host repo with `gitignore check` and
`gitignore sync` for a single registered source.

**Scope:**

- Establish the Go module and conventional project layout.
- Add the `cubby` executable entry point.
- Add a `Makefile` with:
  - `make build`
  - `make test`
  - `make lint`
  - `make check`
- Configure linting through `golangci-lint`.
- Expose the v0.1 command hierarchy in help output.
- Load a host `.cubby.toml` with one registered source.
- Load that source repo's `cubby.toml`.
- Implement enough profile discovery to compute the union of declared profiles.
- Implement `cubby gitignore check`.
- Implement `cubby gitignore sync`.
- Add an end-to-end test that creates throwaway host/source repos and verifies
  `gitignore check` then `gitignore sync`.

**Verification:**

- `make build` produces a runnable `cubby` binary.
- `make test` runs unit and end-to-end tests.
- `make lint` runs `golangci-lint` successfully.
- `make check` runs build, test, and lint.
- `cubby --help` shows the expected v0.1 command surface.
- `gitignore check` reports missing `*.<profile>.*` and `*.<profile>` patterns
  for profiles declared by the registered source.
- `gitignore sync` appends missing patterns and a second `gitignore check`
  succeeds.

## Milestone 2: Single-Source Link and Unlink Smoke Slice

**Outcome:** The core smoke test from `SPEC.md` passes for one host repo, one
source repo, and one selected profile.

**Scope:**

- Implement `cubby link --profile <name>` for one registered source.
- Match profile files using declared source profiles.
- Preserve source-relative paths and profile suffixes in the host repo.
- Create relative symlinks from the host repo into the source repo.
- Create parent directories as needed.
- Treat a correctly linked file as an idempotent no-op.
- Implement `cubby unlink --profile <name>` for links created by this slice.
- Add an end-to-end test covering:
  - create host/source repos
  - seed `src/nvim/init.work.lua`
  - link `work`
  - assert the host symlink is relative and resolves to the source file
  - run gitignore check/sync
  - unlink `work`
  - assert the symlink is gone

**Verification:**

- The full smoke test from `SPEC.md` passes.
- Re-running `link` over an existing correct symlink succeeds without changes.
- `unlink` removes the selected profile symlink.
- `unlink` leaves regular host files alone.

## Milestone 3: Profile Selection and Discovery Slice

**Outcome:** The working single-source flow supports realistic profile
selection and file discovery rules.

**Scope:**

- Support repeatable `--profile` flags.
- Support optional CSV profile input.
- Support `$CUBBY_PROFILE` as fallback.
- Ensure flag values override environment values.
- Error when profile-scoped commands receive no profile selection.
- Recognize both valid profile filename forms:
  - `*.<profile>.*`
  - `*.<profile>`
- Exclude the unsupported literal `.<profile>` filename form.
- Ignore lookalike files for undeclared profiles.
- Apply source `ignore` rules.
- Implement `cubby profile list`.
- Add end-to-end tests for flag selection, environment selection, multi-profile
  selection, ignored files, and undeclared-profile lookalikes.

**Verification:**

- `link` and `unlink` accept profiles from flags and environment variables.
- Flag-provided profiles win over `$CUBBY_PROFILE`.
- Multi-profile invocation links and unlinks only selected profile files.
- Files for undeclared profiles are ignored.
- Ignored source files are not linked.
- `profile list` prints the union of declared profiles.

## Milestone 4: Conflict and Safety Slice

**Outcome:** `cubby link` is safe to run in a real host repo because conflicts
are detected, reported, and never resolved by deleting user files.

**Scope:**

- Detect a regular file already present at the projected host path.
- Detect an unexpected symlink already present at the projected host path.
- Treat cross-source collisions as conflicts once multi-source config appears
  in this slice's tests.
- Honor per-source `ignore_conflicts`.
- Add CLI conflict skipping through `--ignore-conflicts` or the final selected
  flag name.
- Ensure conflict skipping links non-conflicting files and reports skipped
  files.
- Add end-to-end tests for regular-file conflicts, unexpected-symlink
  conflicts, skipped conflicts, and idempotent correct symlinks.

**Verification:**

- Default conflicts exit non-zero.
- Existing regular files are never overwritten.
- Unexpected symlinks are never replaced.
- Correct symlinks remain idempotent no-ops.
- Conflict-skipping mode exits successfully when only skippable conflicts are
  encountered and reports what was skipped.

## Milestone 5: Multi-Source Inventory Slice

**Outcome:** The core link/unlink/gitignore flow works across multiple
registered sources, and source inventory commands are useful.

**Scope:**

- Load multiple registered sources from `.cubby.toml`.
- Enforce host `profiles` as strict opt-in per source.
- Implement `cubby source list`.
- Expand `cubby profile list` across all registered sources.
- Ensure `gitignore check` and `gitignore sync` use the union of profiles from
  all registered sources.
- Ensure `link` and `unlink` apply selected profiles across all eligible
  sources.
- Report host-requested profiles that a source does not declare as diagnostics,
  not link-time hard errors.
- Add end-to-end tests for multi-source linking, source collisions, strict
  profile opt-in, and gitignore union behavior.

**Verification:**

- `source list` prints registered sources.
- `profile list` prints the union of declared profiles.
- Sources with omitted or empty host `profiles` do not link files.
- Multi-source `link` creates the expected relative symlinks.
- Multi-source `unlink` removes only matching managed links.
- Missing source-declared profiles are visible in diagnostics without blocking
  unrelated link work.

## Milestone 6: Status, Doctor, and Prune Slice

**Outcome:** `cubby` can explain and clean the state it created, without using
an active-profile state file.

**Scope:**

- Discover managed symlinks by walking the host repo and resolving targets into
  registered source repos.
- Implement `cubby status`.
- Report linked files, source ownership, profile ownership, and drift.
- Implement `cubby prune`.
- Remove dangling managed symlinks whose targets no longer exist.
- Implement `cubby doctor` as an aggregate health check.
- Include diagnostics for missing sources, missing gitignore patterns,
  conflicts, dangling managed symlinks, and missing requested profiles.
- Add end-to-end tests that create real host/source drift and validate
  `status`, `doctor`, and `prune` behavior.

**Verification:**

- `status` reports linked files with source and profile information.
- `status` identifies drift between host symlinks and source files.
- `doctor` exits zero for a healthy setup.
- `doctor` exits non-zero and reports unhealthy setup issues.
- `prune` removes dangling managed symlinks.
- `prune` leaves valid symlinks and unmanaged symlinks untouched.

## Milestone 7: Source-Scoped Lazygit Slice

**Outcome:** Source-scoped command selection is proven through
`cubby lazygit`.

**Scope:**

- Implement `cubby lazygit [--source <name>]`.
- Select the only registered source implicitly.
- Require `--source` when multiple sources are registered.
- Error clearly for unknown source names.
- Run `lazygit` in the selected source repo.
- Add end-to-end or command-runner tests for single-source selection,
  multi-source ambiguity, explicit source selection, and missing `lazygit`.

**Verification:**

- A single registered source is selected implicitly.
- Multiple registered sources require `--source`.
- Unknown source names produce a clear error.
- The command launches `lazygit` from the selected source repo.

## Milestone 8: Distribution and Release Readiness Slice

**Outcome:** The v0.1 command surface is covered by end-to-end acceptance tests
and can be installed from source.

**Scope:**

- Harden the end-to-end test suite around the full v0.1 workflow.
- Ensure `go install` works.
- Prepare the Homebrew `--HEAD` installation path.
- Confirm output remains scriptable and pipe-friendly while using Lipgloss only
  for terminal styling.
- Document the supported v0.1 behavior and known non-goals.

**Verification:**

- `make check` passes.
- End-to-end tests cover:
  - gitignore check/sync
  - link/unlink
  - conflicts
  - multi-source repos
  - multi-profile invocations
  - env-var-only profile selection
  - source selection errors
  - ignored source files
  - dangling symlink pruning
  - doctor diagnostics
- `go install` installs a working `cubby` command.
- The Homebrew `--HEAD` formula can build from the main branch.
