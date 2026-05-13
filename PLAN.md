# cubby Implementation Plan

This plan turns `SPEC.md` into vertical, testable slices. Each milestone should
produce a knowably working behavior that crosses the real CLI, config loading,
filesystem state, and command exit codes. Shared internals can grow as needed,
but the plan favors end-to-end proof over building hidden layers first.

## [x] Milestone 1: Tooling, CLI Shell, and First End-to-End Guard

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
- Load a host `.cubby.toml` from the current directory with one registered source; `cubby` uses Stow-like ergonomics and must be run from the host repo root.
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

## [x] Milestone 2: Single-Source Link and Unlink Smoke Slice

**Outcome:** The core smoke test from `SPEC.md` passes for one host repo, one
source repo, and one explicitly selected profile.

**Guiding decisions:**

- `link` and `unlink` require `--profile` in this slice. They are
  profile-scoped commands and must not infer "all profiles". `$CUBBY_PROFILE`
  fallback is deferred to Milestone 3.
- Profile filenames are matched by exact dot segment, not fuzzy substring:
  `*.<profile>.*` and `*.<profile>` are valid; bare `.<profile>` is not.
- Multi-dot files are preserved as-is. For profile `work`,
  `archive.work.tar.gz` links to `archive.work.tar.gz`.
- Dotfiles work when the profile is a suffix/segment. For profile `work`,
  `.zshrc.work`, `.zshrc.work.local`, and `.gitignore.work` are valid, while
  `.work` is unsupported.
- Cubby preserves source-relative paths and profile suffixes. It does not turn
  `.zshrc.work` into `.zshrc`; downstream host tooling remains responsible for
  consuming profile-scoped files.

**Phase 1: Add the end-to-end smoke test first**

- Add an e2e test that creates throwaway `host/` and `src/` directories.
- Seed `src/cubby.toml` with `profiles = ["work"]`.
- Seed `host/.cubby.toml` with one registered source and `profiles = ["work"]`.
- Seed `src/nvim/init.work.lua`.
- Run `cubby link --profile work` from the host root.
- Assert `host/nvim/init.work.lua` exists, is a relative symlink, and resolves
  to `src/nvim/init.work.lua`.
- Run `cubby link --profile work` again and assert it succeeds as an
  idempotent no-op.
- Run the existing gitignore check/sync flow.
- Run `cubby unlink --profile work` and assert the symlink is gone.
- Add a regular host file at a projected profile path and assert `unlink`
  leaves it alone.

**Phase 2: Add profile flag handling for `link` and `unlink`**

- Replace the placeholder profile commands with real Cobra commands.
- Add a small command helper that reads `--profile` values.
- Require at least one non-empty `--profile` value for now.
- Trim empty values and reject an effectively empty selection.
- Allow Cobra's current string-slice behavior, but defer full repeated flag,
  CSV, and env-var semantics to Milestone 3.

**Phase 3: Add profile-file discovery**

- Add a small internal package for discovering source profile files.
- Walk the source repo with `filepath.WalkDir`.
- Skip directories, `.git/`, and the source `cubby.toml` config.
- Consider regular source files only. Source symlinks are not projected, and
  directory symlinks are not a supported link target.
- Match against declared source profiles only.
- Match basenames using the valid forms:
  - `*.<profile>.*`, e.g. `nvim/init.work.lua`, `archive.work.tar.gz`
  - `*.<profile>`, e.g. `Makefile.work`, `.gitignore.work`, `.zshrc.work`
- Do not match the unsupported literal `.<profile>` basename, e.g. `.work`.
- Avoid substring false positives such as `homework` matching profile `work` or
  `workbench` matching profile `work`.
- Return source-relative paths so link/unlink can project them into the host
  repo unchanged.

**Phase 4: Implement `cubby link --profile <name>`**

- Load the project via `config.LoadProject()`.
- For this slice, operate on the single registered source used by the smoke
  test while keeping the implementation naturally iterable over sources.
- Skip a selected profile for a source if the source does not declare it.
- Discover matching profile files for the selected profile.
- For each match:
  - Compute the source file path from the source root and source-relative path.
  - Compute the host path from the host root and the same relative path.
  - Create parent directories in the host repo as needed.
  - Compute a relative symlink target with `filepath.Rel(filepath.Dir(hostPath), sourcePath)`.
  - Create the symlink if the host path is absent.
  - Treat an existing symlink that resolves to the same source file as an
    idempotent no-op.
  - Return an error for an existing regular file or unexpected symlink; never
    overwrite user files.

**Phase 5: Implement `cubby unlink --profile <name>`**

- Load the project and discover matching selected-profile source files.
- For each projected host path:
  - Missing path: no-op.
  - Regular file: leave it alone.
  - Correct symlink to the expected source file: remove it.
  - Unexpected symlink: leave it alone in this slice.
- Do not remove parent directories unless a later milestone explicitly needs
  empty-directory cleanup.

**Phase 6: Add focused unit tests**

- Verify `init.work.lua` matches profile `work`.
- Verify `archive.work.tar.gz` matches profile `work`.
- Verify `Makefile.work`, `.gitignore.work`, and `.zshrc.work` match profile
  `work`.
- Verify `.work` does not match.
- Verify `script.workbench.sh` and `thing.homework` do not match profile
  `work`.
- Verify lookalikes for undeclared profiles are ignored.
- Verify nested source-relative paths are preserved.
- Verify relative symlink target computation for nested host paths.
- Verify an existing correct symlink is detected as idempotent.

**Verification:**

- The full smoke test from `SPEC.md` passes.
- `cubby link --profile work` creates relative symlinks into the source repo.
- Re-running `link` over an existing correct symlink succeeds without changes.
- `cubby unlink --profile work` removes the selected profile symlink.
- `unlink` leaves regular host files alone.
- Existing Milestone 1 gitignore tests still pass.
- `make test` passes.
- `make lint` passes.
- `make check` passes.

**Deferred to later milestones:**

- `$CUBBY_PROFILE` fallback and flag-over-env precedence.
- Complete repeated `--profile` and CSV multi-profile semantics.
- Source `ignore` rules.
- Conflict skipping via config or CLI flag.
- Multi-source behavior and cross-source collisions.
- Status/prune/doctor state discovery.

## Milestone 3: Profile Selection and Discovery Slice

**Outcome:** The working single-source flow supports realistic profile
defaults, flag/env selection, source-declared profile availability, source
ignore rules, and scriptable profile discovery output.

**Scope:**

- Migrate the host config schema from per-source host profiles to top-level host
  profile defaults:

  ```toml
  profiles = ["work", "personal"]

  [[source]]
  name = "src"
  path = "../src"
  ```

- Keep source `cubby.toml` profiles as the source of truth for what each source
  provides.
- Use `go-config-loader` for command profile selection with precedence:

  ```text
  defaults < host .cubby.toml profiles < CUBBY_PROFILE < --profile
  ```

- Support repeatable `--profile` flags and CSV profile input for both flags and
  `$CUBBY_PROFILE`.
- Ensure flag values override environment and host-default values completely.
- Treat host `profiles` as defaults, not an allowlist; flags/env may select
  profiles not listed in host defaults.
- Error when flags/env/host defaults produce no effective profile selection.
- Error before link/unlink side effects when any selected profile is declared by
  no registered source.
- Per source, apply only the intersection of the effective selection and that
  source's declared profiles.
- Recognize both valid profile filename forms:
  - `*.<profile>.*`
  - `*.<profile>`
- Exclude the unsupported literal `.<profile>` filename form.
- Ignore lookalike files for undeclared profiles.
- Apply source `ignore` rules using source-relative `/`-normalized paths,
  basename matching for patterns without `/`, and doublestar-style `**` globs.
- Error on invalid source ignore patterns.
- Implement `cubby profile list` as sorted, one-profile-per-line output from
  the union of source-declared profiles only.
- Keep `gitignore check` and `gitignore sync` using the union of all
  source-declared profiles for safety.
- Add focused end-to-end tests for host-default selection, flag selection,
  environment selection, flag-over-env precedence, multi-profile selection,
  no-selection errors, unknown-profile errors before side effects, ignored
  files, undeclared-profile lookalikes, and `profile list` output.
- Add table-driven tests that deliberately distinguish host defaults, effective
  command selection, and source-declared profiles so bugs that mix up these
  profile sets are caught.
- Add an explicit empty-flag test, preferably with `--profile=`, proving that a
  changed empty flag produces an empty effective selection and does not fall
  back to env or host defaults.

**Verification:**

- Existing Milestone 1 and 2 behaviors still pass under the new host schema.
- `link` and `unlink` accept profiles from host defaults, `$CUBBY_PROFILE`, and
  `--profile` with the expected precedence.
- Flag-provided profiles win over `$CUBBY_PROFILE` and host defaults.
- Multi-profile invocation links and unlinks only selected profile files.
- A selected profile unknown to all sources fails before filesystem changes.
- Files for undeclared profiles are ignored.
- Ignored source files are not linked.
- Invalid source ignore patterns fail clearly.
- `profile list` prints the sorted union of source-declared profiles.
- `gitignore check` and `gitignore sync` check/sync patterns for every
  source-declared profile.

## Milestone 4: Conflict and Safety Slice

**Outcome:** `cubby link` is safe to run in a real host repo because conflicts
are detected, reported, and never resolved by deleting user files.

**Scope:**

- Detect a regular file already present at the projected host path.
- Detect an unexpected symlink already present at the projected host path.
- Treat cross-source collisions as conflicts once multi-source config appears
  in this slice's tests.
- Honor top-level host `ignore_conflicts` as the host-wide conflict-skip default.
- Add CLI conflict skipping through `--ignore-conflicts` or the final selected
  flag name.
- Add `--dry-run` for `link` and `unlink` so users can preview planned creates,
  removals, skips, and conflicts without mutating the filesystem.
- Ensure conflict skipping links non-conflicting files and reports skipped
  files.
- Add end-to-end tests for regular-file conflicts, unexpected-symlink
  conflicts, skipped conflicts, idempotent correct symlinks, and dry-run
  non-mutation behavior.

**Verification:**

- Default conflicts exit non-zero.
- Existing regular files are never overwritten.
- Unexpected symlinks are never replaced.
- Correct symlinks remain idempotent no-ops.
- Conflict-skipping mode exits successfully when only skippable conflicts are
  encountered and reports what was skipped.
- Dry-run mode reports the same planned work/conflicts without creating or
  removing links.

## Milestone 5: Multi-Source Inventory Slice

**Outcome:** The core link/unlink/gitignore flow works across multiple
registered sources, and source inventory commands are useful.

**Scope:**

- Load multiple registered sources from `.cubby.toml`.
- Use top-level host `profiles` as default command selection across all
  registered sources.
- Implement `cubby source list`.
- Ensure `cubby profile list` remains correct across all registered sources.
- Ensure `gitignore check` and `gitignore sync` use the union of profiles from
  all registered sources.
- Ensure `link` and `unlink` apply effective selected/default profiles across
  all eligible sources.
- Treat a selected profile that is missing from one source as a diagnostic, not
  a hard error, as long as at least one registered source declares it.
- Add end-to-end tests for multi-source linking, source collisions, top-level
  host default selection, explicit profile selection, and gitignore union
  behavior.

**Verification:**

- `source list` prints registered sources.
- `profile list` prints the union of declared source profiles.
- Empty top-level host profiles require explicit flag/env profile selection.
- Multi-source `link` creates the expected relative symlinks.
- Multi-source `unlink` removes only matching managed links.
- Profiles missing from a given source are visible in diagnostics without
  blocking unrelated link work when another source provides the selected
  profile.

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
  - dry-run link/unlink previews
  - multi-source repos
  - multi-profile invocations
  - env-var-only profile selection
  - source selection errors
  - ignored source files
  - dangling symlink pruning
  - doctor diagnostics
- `go install` installs a working `cubby` command.
- The Homebrew `--HEAD` formula can build from the main branch.
